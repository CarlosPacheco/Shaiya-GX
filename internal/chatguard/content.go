// Package chatguard inspects chat text locally. It treats URLs as data and
// never resolves, fetches or opens a destination.
package chatguard

import (
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/unicode/norm"
)

type Result struct {
	Text          string
	Quiet         bool
	Impersonation bool
}

const blockedLink = "[link blocked]"

var (
	chatToken       = regexp.MustCompile(`[^\s\p{Z}<>\x22\x27]+`)
	roleClaim       = regexp.MustCompile(`^(?:\s*[\[(]\s*(?:gm|admin|staff|support|soporte)\s*[\])]|(?:gm|admin|staff|support|soporte)\s*[:：]|(?:soy|somos|i am|i'm|we are)\s+(?:(?:un|el|la|a|an|the)\s+)?(?:gm|admin(?:istrador)?|staff|soporte|support)\b|(?:soporte|support|staff)\s+(?:oficial|official|gx)\b)`)
	secret          = regexp.MustCompile(`\b(?:password|passcode|contrasena|clave|token|2fa|otp|codigo|verification code|recovery code)\b`)
	request         = regexp.MustCompile(`\b(?:send|share|enter|provide|verify|need|envia|enviame|manda|mandame|pasa|pasame|dame|decime|dime|ingresar?|verifica|confirma)\b`)
	download        = regexp.MustCompile(`\b(?:download|install|run|descarga|descargar|instala|instalar|ejecuta|ejecutar)\b`)
	linkPunctuation = strings.NewReplacer("[.]", ".", "(.)", ".", "[dot]", ".", "(dot)", ".", "[:]", ":", "[: ]", ":", "\\/", "/", "。", ".", "｡", ".")
)

// CleanText keeps ordinary Unicode and emoji shaping. It removes invisible
// direction overrides, flattens line breaks, and limits extreme glyph runs.
func CleanText(text string) string {
	var out strings.Builder
	var previous rune
	run, marks, length := 0, 0, 0
	for _, r := range text {
		if unsafeFormat(r) {
			continue
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp) {
			r = ' '
		}
		if unicode.IsMark(r) {
			marks++
			if marks > 6 {
				continue
			}
		} else {
			marks = 0
		}
		if r == previous {
			run++
			if run >= 12 {
				continue
			}
		} else {
			previous, run = r, 0
		}
		if length == 180 {
			out.WriteRune('…')
			break
		}
		out.WriteRune(r)
		length++
	}
	return out.String()
}

func unsafeFormat(r rune) bool {
	if r == '\u200c' || r == '\u200d' || r >= '\U000e0020' && r <= '\U000e007f' {
		return false // Script joiners and emoji subdivision flags.
	}
	return unicode.Is(unicode.Cf, r)
}

const linkDelimiters = ".:/\\[(。｡．：／＼［（"

func inspectionText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || r == '\ufe0e' || r == '\ufe0f' {
			return -1
		}
		return unicode.ToLower(r)
	}, norm.NFKC.String(text))
}

// Filter hides a credential or download lure using an unverified staff claim.
// Other suspicious messages retain their readable portion and redact links.
func Filter(text string, staff bool) Result {
	// Inspect the full content before applying display limits or modifying URLs.
	clean := strings.Map(func(r rune) rune {
		if unsafeFormat(r) {
			return -1
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp) {
			return ' '
		}
		return r
	}, text)
	inspected := inspectionText(clean)
	if !staff && roleClaim.MatchString(inspected) {
		words := strings.Map(func(r rune) rune {
			if unicode.IsMark(r) {
				return -1
			}
			return r
		}, norm.NFD.String(inspected))
		if secret.MatchString(words) && request.MatchString(words) || download.MatchString(words) {
			return Result{Quiet: true, Impersonation: true}
		}
	}
	var redacted strings.Builder
	start := 0
	quiet := clean != text
	if !strings.ContainsAny(clean, linkDelimiters) {
		return Result{Text: CleanText(clean), Quiet: quiet}
	}
	for _, span := range chatToken.FindAllStringIndex(clean, -1) {
		word := clean[span[0]:span[1]]
		candidate := strings.TrimLeft(word, "([{`«“‘")
		candidate = trimLinkSuffix(candidate)
		if !dangerousLink(candidate) {
			continue
		}
		prefix := len(word) - len(strings.TrimLeft(word, "([{`«“‘"))
		redacted.WriteString(clean[start : span[0]+prefix])
		redacted.WriteString(blockedLink)
		start = span[0] + prefix + len(candidate)
		quiet = true
	}
	if start != 0 {
		redacted.WriteString(clean[start:])
		clean = redacted.String()
	}
	return Result{Text: CleanText(clean), Quiet: quiet}
}

func trimLinkSuffix(token string) string {
	for {
		token = strings.TrimRight(token, "`»”’.,!;")
		if len(token) == 0 {
			return token
		}
		open := ""
		switch token[len(token)-1] {
		case ')':
			open = "("
		case ']':
			open = "["
		case '}':
			open = "{"
		}
		if open == "" || strings.Count(token, token[len(token)-1:]) <= strings.Count(token, open) {
			return token
		}
		token = token[:len(token)-1]
	}
}

func dangerousLink(token string) bool {
	if !strings.ContainsAny(token, linkDelimiters) {
		return false
	}
	value := linkPunctuation.Replace(inspectionText(token))
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, `\\`) || len(value) > 2 && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	for _, prefix := range []string{"/home/", "/etc/", "/tmp/", "/users/", "/var/", "/mnt/"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	if strings.HasPrefix(value, "hxxp://") || strings.HasPrefix(value, "hxxps://") {
		value = "http" + value[4:]
	}
	explicit := strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
	if !explicit {
		if scheme, _, found := strings.Cut(value, ":"); found {
			switch scheme {
			case "file", "javascript", "data", "vbscript", "blob", "shell", "smb", "ftp", "sftp", "ssh", "steam", "discord", "intent", "magnet", "mailto", "tel", "sms", "tg", "whatsapp", "vscode", "zoommtg", "chrome", "chrome-extension":
				return true
			}
			if strings.Contains(value, "://") || strings.HasPrefix(scheme, "ms-") {
				return true
			}
		}
		networkPath := strings.HasPrefix(value, "//")
		if !networkPath && (strings.Contains(value, "@") || !strings.Contains(value, ".") || strings.HasPrefix(value, "/")) {
			return false
		}
		explicit = networkPath
		value = "https://" + strings.TrimPrefix(value, "//")
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" {
		return explicit
	}
	host := strings.TrimSuffix(u.Hostname(), ".")
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() {
			return true
		}
	} else {
		ascii, err := idna.Lookup.ToASCII(host)
		if err != nil {
			return explicit
		}
		if !explicit {
			if _, icann := publicsuffix.PublicSuffix(ascii); !icann && !isLocalDomain(ascii) {
				return false
			}
		}
		tld := ascii[strings.LastIndexByte(ascii, '.')+1:]
		if !strings.Contains(ascii, ".") || strings.Trim(tld, "0123456789") == "" || isLocalDomain(ascii) || impersonatesDomain(ascii) {
			return true
		}
	}
	if u.User != nil {
		return true
	}
	filename := strings.ToLower(u.Path)
	if decoded, err := url.PathUnescape(filename); err == nil {
		filename = strings.ToLower(decoded)
	}
	switch path.Ext(strings.TrimRight(filename, " .")) {
	case ".exe", ".msi", ".msix", ".scr", ".com", ".bat", ".cmd", ".ps1", ".vbs", ".js", ".jse", ".hta", ".sh", ".appimage", ".apk", ".dmg", ".pkg", ".reg", ".lnk", ".jar":
		return true
	}
	return false
}

func isLocalDomain(host string) bool {
	return strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".localhost")
}

// These are project distribution and community brands. Lookalike matching is
// scoped to those brands so unrelated international domain names stay readable.
func impersonatesDomain(ascii string) bool {
	for _, domain := range []string{"github.com", "discord.com", "discord.gg", "discordapp.com", "discordapp.net", "drive.google.com"} {
		if ascii == domain || strings.HasSuffix(ascii, "."+domain) {
			return false
		}
	}
	host, _ := idna.Lookup.ToUnicode(ascii)
	host = strings.Map(func(r rune) rune {
		if similar, ok := brandLookalikes[r]; ok {
			return similar
		}
		return r
	}, inspectionText(host))
	labels := strings.Split(host, ".")
	for i, label := range labels {
		if label == "github" || label == "discord" || label == "drive" && i+1 < len(labels) && labels[i+1] == "google" || strings.Contains(strings.ReplaceAll(label, "-", ""), "shaiyagx") {
			return true
		}
		for _, brand := range []string{"github-", "discord-"} {
			if rest, ok := strings.CutPrefix(label, brand); ok {
				switch rest {
				case "login", "verify", "signin", "support", "account", "security":
					return true
				}
			}
		}
	}
	return false
}

var brandLookalikes = map[rune]rune{
	'а': 'a', 'α': 'a', 'е': 'e', 'ε': 'e', 'і': 'i', 'ι': 'i', 'һ': 'h', 'ј': 'j', 'к': 'k', 'κ': 'k',
	'о': 'o', 'ο': 'o', 'ρ': 'p', 'р': 'p', 'с': 'c', 'ϲ': 'c', 'ѕ': 's', 'т': 't', 'τ': 't',
	'х': 'x', 'χ': 'x', 'у': 'y', 'υ': 'y', 'ԁ': 'd', 'ɡ': 'g', 'ӏ': 'l', '0': 'o', '1': 'i',
}

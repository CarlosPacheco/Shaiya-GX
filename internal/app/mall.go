package app

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// The gift claim packet gives only two slots: the original item definition
// supplies the initial quality and maximum stack after server confirmation.
func (a *App) confirmedMall(packet []byte) {
	s := a.state
	if proto.Op(packet) == a.mall.pending {
		a.mall.pending = 0
	}
	switch proto.Op(packet) {
	case proto.PointBuy:
		result := s.Mall.LastPurchase
		messages := [...]string{a.tr(12349), a.tr(12350), a.tr(12351), a.tr(12352)}
		message := a.tr(12353)
		if int(result.Result) < len(messages) {
			message = messages[result.Result]
		}
		a.questMessage(message)
	case proto.PointGiftSend:
		result := s.Mall.LastGift
		messages := map[uint8]string{0: a.tr(12354), 1: a.tr(12355), 2: a.tr(12356), 3: a.tr(12357), 4: a.tr(12358), 5: a.tr(12359), 6: a.tr(12360), 10: a.tr(12351)}
		message := messages[result.Result]
		if message == "" {
			message = a.tr(12361)
		}
		a.questMessage(message)
	case proto.PointStoredItemGet:
		claim := s.Mall.PendingClaim
		if claim == nil {
			return
		}
		def := a.itemsItem(claim.Gift.Type, claim.Gift.TypeID)
		if def == nil {
			log.Printf("mall: missing item definition %d/%d after confirmed gift claim", claim.Gift.Type, claim.Gift.TypeID)
			a.questMessageID(12137)
			return
		}
		if s.ResolveMallClaim(min(def.Quality, 4999), max(def.Count, 1)) {
			a.questMessageID(12138)
		}
	}
}

// Native wire requests are exposed while the mall artwork is integrated.
// Purchases and claims wait for Rapture's replies before changing state.
func (a *App) mallCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/mall") {
		return false
	}
	if len(fields) == 1 {
		a.openMall()
		return true
	}
	var packet []byte
	if len(fields) >= 2 && strings.EqualFold(fields[1], "catalog") {
		page := 1
		if len(fields) > 3 {
			a.questMessageID(12139)
			return true
		}
		if len(fields) == 3 {
			var err error
			page, err = strconv.Atoi(fields[2])
			if err != nil || page < 1 {
				a.questMessageID(12139)
				return true
			}
		}
		if a.mallCatalog == nil {
			if err := a.loadMallCatalog(); err != nil {
				log.Printf("Item Mall catalog: %v", err)
				a.questMessageID(12140)
				return true
			}
		}
		visible := []cash.Product{}
		for _, product := range a.mallCatalog {
			if product.Bag != 0 {
				visible = append(visible, product)
			}
		}
		const pageSize = 5
		pages := max(1, (len(visible)+pageSize-1)/pageSize)
		if page > pages {
			a.questMessage(a.msg.Format(12304, map[string]string{"v": fmt.Sprint(pages)}))
			return true
		}
		a.questMessage(a.msg.Format(12305, map[string]string{"v": fmt.Sprint(page), "v2": fmt.Sprint(pages)}))
		for _, product := range visible[(page-1)*pageSize : min(page*pageSize, len(visible))] {
			a.questMessage(a.msg.Format(12453, map[string]string{"v": product.Code, "p": a.mallProductText(product).Name, "b": fmt.Sprint(product.Cost)}))
		}
		return true
	} else if len(fields) == 2 && strings.EqualFold(fields[1], "points") {
		packet = proto.MallPointsPacket()
	} else if len(fields) == 3 && strings.EqualFold(fields[1], "buy") && len(proto.EncodeText(fields[2])) <= 20 {
		packet = proto.MallBuyPacket(fields[2])
	} else if len(fields) == 4 && strings.EqualFold(fields[1], "gift") && len(proto.EncodeText(fields[2])) <= 20 && len(proto.EncodeText(fields[3])) <= 20 {
		packet = proto.MallGiftPacket(fields[2], fields[3])
	} else if len(fields) == 3 && strings.EqualFold(fields[1], "claim") {
		slot, err := strconv.ParseUint(fields[2], 10, 8)
		if err == nil && slot < 240 && a.state.Mall.PendingClaim == nil {
			if _, ok := a.state.Mall.Stored[uint8(slot)]; ok {
				packet = proto.MallClaimPacket(uint8(slot))
			}
		}
	} else if len(fields) == 2 && strings.EqualFold(fields[1], "gifts") {
		if !a.state.Mall.StoredReady {
			a.questMessageID(12141)
			return true
		}
		if len(a.state.Mall.Stored) == 0 {
			a.questMessageID(12142)
			return true
		}
		for slot := 0; slot < 240; slot++ {
			if gift, ok := a.state.Mall.Stored[uint8(slot)]; ok {
				name := fmt.Sprintf("%d/%d", gift.Type, gift.TypeID)
				if def := a.itemsItem(gift.Type, gift.TypeID); def != nil {
					name = def.Name
				}
				a.questMessage(a.msg.Format(12306, map[string]string{"v": fmt.Sprint(slot), "v2": name, "v3": fmt.Sprint(gift.Count)}))
			}
		}
		return true
	}
	if packet == nil {
		a.questMessageID(12143)
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLoginID(12136)
	}
	return true
}

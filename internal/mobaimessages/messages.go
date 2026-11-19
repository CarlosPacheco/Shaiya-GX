// Package mobaimessages stores the indexed dialogue used by GX World mob AI.
package mobaimessages

import (
	"context"
	"database/sql"
)

const Table = "PS_GameDefs_MobAIMessages"

// MigrationSQL seeds the original EP4.5 dialogue with its script indexes.
const MigrationSQL = `CREATE TABLE PS_GameDefs_MobAIMessages(
    "Index" INTEGER PRIMARY KEY CHECK("Index" BETWEEN 0 AND 65535),
    "Text" TEXT NOT NULL
);
INSERT INTO PS_GameDefs_MobAIMessages ("Index", "Text") VALUES
(1,'이단자들에게 철퇴를!'),
(2,'Li laka Amon-Ra'),
(3,'뼈와 살을 분리해주마!!'),
(4,'배신자들에게 피와 죽음의 향연을!!'),
(5,'죽어라! 어리석은 것'),
(6,'네 실력을 한 번 보자'),
(7,'죽음의 시간이다'),
(8,'누구도 죽음을 피해가지는 못한다'),
(9,'저승길을 재촉하는구나!!'),
(10,'흐흐흐~ 오랫만에 고기맛을 보겠군.'),
(11,'과거의 잘못은 너희들의 죽음으로 보상 받으리라~'),
(12,'우리들의 고통을 너희들도 느껴보아라~'),
(13,'피와 광기의 폭풍을!!!'),
(14,'헤..헬투스님들의 여..영광을 위해!!'),
(15,'이..이곳의 대..대장은 나..나다!!'),
(16,'너..너를 죽일거야'),
(17,'꿈의 세계에 온 것을 환영한다'),
(18,'신에게 대항하다니! 어리석은 것들~'),
(19,'이것이 끝이라고 생각하면 오산이다!!'),
(20,'게헤나의 불속에 부활할 것이다'),
(21,'여기까지 날 몰아붙인 것은 칭찬해주지. 그러나 유희는 여기까지다!!'),
(22,'피의 축제에 온 것을 환영한다. 어리석은 피조물들아~'),
(23,'벌레들과 상대하려니 피곤하군. 좀 쉬어볼까'),
(24,'불의 힘이 나를 감싸리라'),
(25,'나를 경배하라! 에테인의 피조물들아'),
(26,'앱소션 쉴드 전개'),
(27,'물의 힘이 나를 보호하리라'),
(28,'대지의 견고함이 나를 보호할 것이다'),
(29,'바람이여~ 나를 감싸라'),
(30,'모든 것은 무로 돌아갈 것이다.'),
(31,'너희들은 넘지말아야할 선을 넘어섰다. 보아라! 너희들의 하찮은 힘을...');
`

// Message is the text associated with an AI script index.
type Message struct {
	Index uint16
	Text  string
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]Message, error) {
	rows, err := db.QueryContext(ctx, `SELECT "Index", "Text" FROM `+Table+` ORDER BY "Index"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		var message Message
		if err = rows.Scan(&message.Index, &message.Text); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

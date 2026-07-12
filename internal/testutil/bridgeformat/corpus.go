package bridgeformat

// Case defines one source-markdown expectation for both bridge dialects.
type Case struct {
	Name              string
	Input             string
	Slack             string
	Telegram          string
	TelegramPlain     string
	TelegramParseMode string
}

// Cases returns the shared outbound-markdown behavior corpus.
func Cases() []Case {
	return []Case{
		{
			Name:              "Should convert bold links and platform control characters",
			Input:             "**bold** [doc](https://x.y) & <tag>",
			Slack:             "*bold* <https://x.y|doc> &amp; &lt;tag&gt;",
			Telegram:          "*bold* [doc](https://x.y) & <tag\\>",
			TelegramPlain:     "**bold** [doc](https://x.y) & <tag>",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should convert italic and strikethrough without crossing lines",
			Input:             "*italic* and ~~gone~~\n* list item",
			Slack:             "_italic_ and ~gone~\n* list item",
			Telegram:          "_italic_ and ~gone~\n\\* list item",
			TelegramPlain:     "*italic* and ~~gone~~\n* list item",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should escape every Telegram MarkdownV2 special outside code",
			Input:             "v2.0! #tag + x-y = {z} | [q]",
			Slack:             "v2.0! #tag + x-y = {z} | [q]",
			Telegram:          "v2\\.0\\! \\#tag \\+ x\\-y \\= \\{z\\} \\| \\[q\\]",
			TelegramPlain:     "v2.0! #tag + x-y = {z} | [q]",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should preserve inline code while escaping surrounding prose",
			Input:             "Use `a_b(x)!` now.",
			Slack:             "Use `a_b(x)!` now.",
			Telegram:          "Use `a_b(x)!` now\\.",
			TelegramPlain:     "Use `a_b(x)!` now.",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should preserve fenced code and escape Telegram code delimiters",
			Input:             "```go\nfmt.Println(`x`)\\path\n```",
			Slack:             "```go\nfmt.Println(`x`)\\path\n```",
			Telegram:          "```go\nfmt.Println(\\`x\\`)\\\\path\n```",
			TelegramPlain:     "```go\nfmt.Println(`x`)\\path\n```",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should resume conversion after a balanced code fence",
			Input:             "```go\n**literal**\n```\n**bold**",
			Slack:             "```go\n**literal**\n```\n*bold*",
			Telegram:          "```go\n**literal**\n```\n*bold*",
			TelegramPlain:     "```go\n**literal**\n```\n**bold**",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should preserve Slack entities and escape them as Telegram prose",
			Input:             "Hey <@U123>, see <https://x.y|docs> and <!here>",
			Slack:             "Hey <@U123>, see <https://x.y|docs> and <!here>",
			Telegram:          "Hey <@U123\\>, see <https://x\\.y\\|docs\\> and <\\!here\\>",
			TelegramPlain:     "Hey <@U123>, see <https://x.y|docs> and <!here>",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should keep pipe tables readable in both dialects",
			Input:             "| A | B |\n|---|---|\n| 1 | 2 |",
			Slack:             "| A | B |\n|---|---|\n| 1 | 2 |",
			Telegram:          "\\| A \\| B \\|\n\\|\\-\\-\\-\\|\\-\\-\\-\\|\n\\| 1 \\| 2 \\|",
			TelegramPlain:     "| A | B |\n|---|---|\n| 1 | 2 |",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should preserve astral emoji and snake case identifiers",
			Input:             "🙂 my_variable_name",
			Slack:             "🙂 my_variable_name",
			Telegram:          "🙂 my\\_variable\\_name",
			TelegramPlain:     "🙂 my_variable_name",
			TelegramParseMode: "MarkdownV2",
		},
		{
			Name:              "Should fall back to plain text for an unterminated code fence",
			Input:             "```go\nunterminated **bold**",
			Slack:             "```go\nunterminated **bold**",
			Telegram:          "```go\nunterminated **bold**",
			TelegramPlain:     "```go\nunterminated **bold**",
			TelegramParseMode: "",
		},
	}
}

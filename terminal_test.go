package terminal

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
)

var TestFiles = []string{
	"control.sh",
	"curl.sh",
	"cursor-save-restore.sh",
	"docker-pull.sh",
	"homer.sh",
	"npm.sh",
	"pikachu.sh",
	"rustfmt.sh",
	"weather.sh",
}

func loadFixture(t testing.TB, base string, ext string) []byte {
	filename := fmt.Sprintf("fixtures/%s.%s", base, ext)
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not load fixture %s: %v", filename, err)
	}
	return data
}

func base64Encode(stringToEncode string) string {
	return base64.StdEncoding.EncodeToString([]byte(stringToEncode))
}

var rendererTestCases = []struct {
	name     string
	input    string
	expected string
}{
	{
		`input that ends in a newline will not include that newline`,
		"hello\n",
		"hello",
	}, {
		`closes colors that get opened`,
		"he\033[32mllo",
		"he<span class=\"term-fg32\">llo</span>",
	}, {
		`treats multi-byte unicode characters as individual runes`,
		"€€€€€€\b\b\baaa",
		"€€€aaa",
	}, {
		`skips over colors when backspacing`,
		"he\x1b[32m\x1b[33m\bllo",
		"h<span class=\"term-fg33\">llo</span>",
	}, {
		`handles \x1b[m (no parameter) as a reset`,
		"\x1b[36mthis has a color\x1b[mthis is normal now\r\n",
		"<span class=\"term-fg36\">this has a color</span>this is normal now",
	}, {
		`treats \x1b[39m as a reset`,
		"\x1b[36mthis has a color\x1b[39mthis is normal now\r\n",
		"<span class=\"term-fg36\">this has a color</span>this is normal now",
	}, {
		`starts overwriting characters when you \r midway through something`,
		"hello\rb",
		"bello",
	}, {
		`colors across multiple lines`,
		"\x1b[32mhello\n\nfriend\x1b[0m",
		"<span class=\"term-fg32\">hello</span>\n&nbsp;\n<span class=\"term-fg32\">friend</span>",
	}, {
		`allows you to control the cursor forwards`,
		"this is\x1b[4Cpoop and stuff",
		"this is    poop and stuff",
	}, {
		`allows you to jump down further than the bottom of the buffer`,
		"this is great \x1b[1Bhello",
		"this is great\n              hello",
	}, {
		`allows you to control the cursor backwards`,
		"this is good\x1b[4Dpoop and stuff",
		"this is poop and stuff",
	}, {
		`allows you to control the cursor upwards`,
		"1234\n56\x1b[1A78\x1b[B",
		"1278\n56",
	}, {
		`allows you to control the cursor downwards`,
		// creates a grid of:
		// aaaa
		// bbbb
		// cccc
		// Then goes up 2 rows, down 1 row, jumps to the begining
		// of the line, rewrites it to 1234, then jumps back down
		// to the end of the grid.
		"aaaa\nbbbb\ncccc\x1b[2A\x1b[1B\r1234\x1b[1B",
		"aaaa\n1234\ncccc",
	}, {
		`doesn't blow up if you go back too many characters`,
		"this is good\x1b[100Dpoop and stuff",
		"poop and stuff",
	}, {
		`doesn't blow up if you backspace too many characters`,
		"hi\b\b\b\b\b\b\b\bbye",
		"bye",
	}, {
		`\x1b[1K clears everything before it`,
		"hello\x1b[1Kfriend!",
		"     friend!",
	}, {
		`clears everything after the \x1b[0K`,
		"hello\nfriend!\x1b[A\r\x1b[0K",
		"\nfriend!",
	}, {
		`handles \x1b[0G ghetto style`,
		"hello friend\x1b[Ggoodbye buddy!",
		"goodbye buddy!",
	}, {
		`preserves characters already written in a certain color`,
		"  \x1b[90m․\x1b[0m\x1b[90m․\x1b[0m\x1b[0G\x1b[90m․\x1b[0m\x1b[90m․\x1b[0m",
		"<span class=\"term-fgi90\">․․․․</span>",
	}, {
		`replaces empty lines with non-breaking spaces`,
		"hello\n\nfriend",
		"hello\n&nbsp;\nfriend",
	}, {
		`preserves opening colors when using \x1b[0G`,
		"\x1b[33mhello\x1b[0m\x1b[33m\x1b[44m\x1b[0Ggoodbye",
		"<span class=\"term-fg33 term-bg44\">goodbye</span>",
	}, {
		`allows clearing lines below the current line`,
		"foo\nbar\x1b[A\x1b[Jbaz",
		"foobaz",
	}, {
		`doesn't freak out about clearing lines below when there aren't any`,
		"foobar\x1b[0J",
		"foobar",
	}, {
		`allows clearing lines above the current line`,
		"foo\nbar\x1b[A\x1b[1Jbaz",
		"barbaz",
	}, {
		`doesn't freak out about clearing lines above when there aren't any`,
		"\x1b[1Jfoobar",
		"foobar",
	}, {
		`allows clearing the entire scrollback buffer with escape 2J`,
		"this is a big long bit of terminal output\nplease pay it no mind, we will clear it soon\nokay, get ready for a disappearing act...\nand...and...\n\n\x1b[2Jhey presto",
		"hey presto",
	}, {
		`allows clearing the entire scrollback buffer with escape 3J also`,
		"this is a big long bit of terminal output\nplease pay it no mind, we will clear it soon\nokay, get ready for a disappearing act...\nand...and...\n\n\x1b[2Jhey presto",
		"hey presto",
	}, {
		`allows erasing the current line up to a point`,
		"hello friend\x1b[1K!",
		"            !",
	}, {
		`allows clearing of the current line`,
		"hello friend\x1b[2K!",
		"            !",
	}, {
		`doesn't close spans if no colors have been opened`,
		"hello \x1b[0mfriend",
		"hello friend",
	}, {
		`\x1b[K correctly clears all previous parts of the string`,
		"remote: Compressing objects:   0% (1/3342)\x1b[K\rremote: Compressing objects:   1% (34/3342)",
		"remote: Compressing objects:   1% (34&#47;3342)",
	}, {
		`handles reverse linefeed`,
		"meow\npurr\nnyan\x1bMrawr",
		"meow\npurrrawr\nnyan",
	}, {
		`collapses many spans of the same color into 1`,
		"\x1b[90m․\x1b[90m․\x1b[90m․\x1b[90m․\n\x1b[90m․\x1b[90m․\x1b[90m․\x1b[90m․",
		"<span class=\"term-fgi90\">․․․․</span>\n<span class=\"term-fgi90\">․․․․</span>",
	}, {
		`escapes HTML`,
		"hello <strong>friend</strong>",
		"hello &lt;strong&gt;friend&lt;&#47;strong&gt;",
	}, {
		`escapes HTML in color codes`,
		"hello \x1b[\"hellomfriend",
		"hello [&quot;hellomfriend",
	}, {
		`handles background colors`,
		"\x1b[30;42m\x1b[2KOK (244 tests, 558 assertions)",
		"<span class=\"term-fg30 term-bg42\">OK (244 tests, 558 assertions)</span>",
	}, {
		`does not attempt to incorrectly nest CSS in HTML (https://github.com/buildkite/terminal-to-html/issues/36)`,
		"Some plain text\x1b[0;30;42m yay a green background \x1b[0m\x1b[0;33;49mnow this has no background but is yellow \x1b[0m",
		"Some plain text<span class=\"term-fg30 term-bg42\"> yay a green background </span><span class=\"term-fg33\">now this has no background but is yellow </span>",
	}, {
		`handles xterm colors`,
		"\x1b[38;5;169;48;5;50mhello\x1b[0m \x1b[38;5;179mgoodbye",
		"<span class=\"term-fgx169 term-bgx50\">hello</span> <span class=\"term-fgx179\">goodbye</span>",
	}, {
		`handles non-xterm codes on the same line as xterm colors`,
		"\x1b[38;5;228;5;1mblinking and bold\x1b",
		`<span class="term-fgx228 term-fg1 term-fg5">blinking and bold</span>`,
	}, {
		`ignores broken escape characters, stripping the escape rune itself`,
		"hi amazing \x1b[12 nom nom nom friends",
		"hi amazing [12 nom nom nom friends",
	}, {
		`handles colors with 3 attributes`,
		"\x1b[0;10;4m\x1b[1m\x1b[34mgood news\x1b[0;10m\n\neveryone",
		"<span class=\"term-fg34 term-fg1 term-fg4\">good news</span>\n&nbsp;\neveryone",
	}, {
		`ends underlining with \x1b[24`,
		"\x1b[4mbegin\x1b[24m\r\nend",
		"<span class=\"term-fg4\">begin</span>\nend",
	}, {
		`ends bold with \x1b[21`,
		"\x1b[1mbegin\x1b[21m\r\nend",
		"<span class=\"term-fg1\">begin</span>\nend",
	}, {
		`ends bold with \x1b[22`,
		"\x1b[1mbegin\x1b[22m\r\nend",
		"<span class=\"term-fg1\">begin</span>\nend",
	}, {
		`ends crossed out with \x1b[29`,
		"\x1b[9mbegin\x1b[29m\r\nend",
		"<span class=\"term-fg9\">begin</span>\nend",
	}, {
		`ends italic out with \x1b[23`,
		"\x1b[3mbegin\x1b[23m\r\nend",
		"<span class=\"term-fg3\">begin</span>\nend",
	}, {
		`ends decreased intensity with \x1b[22`,
		"\x1b[2mbegin\x1b[22m\r\nend",
		"<span class=\"term-fg2\">begin</span>\nend",
	}, {
		`ignores cursor show/hide`,
		"\x1b[?25ldoing a thing without a cursor\x1b[?25h",
		"doing a thing without a cursor",
	}, {
		`renders simple images on their own line`, // http://iterm2.com/images.html
		"hi\x1b]1337;File=name=MS5naWY=;inline=1:AA==\ahello",
		"hi\n" + `<img alt="1.gif" src="data:image/gif;base64,AA==">` + "\nhello",
	}, {
		`does not start a new line for iterm images if we're already at the start of a line`,
		"\x1b]1337;File=name=MS5naWY=;inline=1:AA==\a",
		`<img alt="1.gif" src="data:image/gif;base64,AA==">`,
	}, {
		`silently ignores unsupported ANSI escape sequences`,
		"abc\x1b]9999\aghi",
		"abcghi",
	}, {
		`correctly handles images that we decide not to render`,
		"hi\x1b]1337;File=name=MS5naWY=;inline=0:AA==\ahello",
		"hihello",
	}, {
		`renders external images`,
		"\x1b]1338;url=http://foo.com/foobar.gif;alt=foo bar\a",
		`<img alt="foo bar" src="http://foo.com/foobar.gif">`,
	}, {
		`disallows non-allow-listed schemes for images`,
		"before\x1b]1338;url=javascript:alert(1);alt=hello\x07after",
		"before\n&nbsp;\nafter", // don't really care about the middle, as long as it's white-spacey
	}, {
		`renders links, and renders them inline on other content`,
		"a link to \x1b]1339;url=http://google.com;content=google\a.",
		`a link to <a href="http://google.com">google</a>.`,
	}, {
		`uses URL as link content if missing`,
		"\x1b]1339;url=http://google.com\a",
		`<a href="http://google.com">http://google.com</a>`,
	}, {
		`protects inline images against XSS by escaping HTML during rendering`,
		"hi\x1b]1337;File=name=" + base64Encode("<script>.pdf") + ";inline=1:AA==\ahello",
		"hi\n" + `<img alt="&lt;script&gt;.pdf" src="data:application/pdf;base64,AA==">` + "\nhello",
	}, {
		`protects external images against XSS by escaping HTML during rendering`,
		"\x1b]1338;url=\"https://example.com/a.gif&a=<b>&c='d'\";alt=foo&bar;width=\"<wat>\";height=2px\a",
		`<img alt="foo&amp;bar" src="https://example.com/a.gif&amp;a=%3Cb%3E&amp;c=%27d%27" width="&lt;wat&gt;em" height="2px">`,
	}, {
		`protects links against XSS by escaping HTML during rendering`,
		"\x1b]1339;url=\"https://example.com/a.gif&a=<b>&c='d'\";content=<h1>hello</h1>\a",
		`<a href="https://example.com/a.gif&amp;a=%3Cb%3E&amp;c=%27d%27">&lt;h1&gt;hello&lt;/h1&gt;</a>`,
	}, {
		`disallows javascript: scheme URLs`,
		"\x1b]1339;url=javascript:alert(1);content=hello\x07",
		`<a href="#">hello</a>`,
	}, {
		`allows artifact: scheme URLs`,
		"\x1b]1339;url=artifact://hello.txt\x07\n",
		`<a href="artifact://hello.txt">artifact://hello.txt</a>`,
	}, {
		`renders bk APC escapes as processing instructions`,
		"\x1b_bk;x=llamas\\;;y=alpacas\x07",
		`<?bk x="llamas;" y="alpacas"?>`,
	}, {
		`renders bk APC escapes as processing instructions`,
		"\x1b" + `_bk;a='1 ("one")';b="2 ('two')"` + "\x07",
		`<?bk a="1 (&#34;one&#34;)" b="2 (&#39;two&#39;)"?>`,
	}, {
		`renders bk APC escapes followed by text`,
		"\x1b_bk;t=123\x07hello",
		`<?bk t="123"?>hello`,
	}, {
		`handles bk APC escapes surrounded by text`,
		"hello \x1b_bk;t=123\x07world",
		`<?bk t="123"?>hello world`,
	}, {
		`prefixes lines with the last timestamp seen`,
		"hello\x1b_bk;t=123\x07 world\x1b_bk;t=456\x07!",
		`<?bk t="456"?>hello world!`,
	}, {
		`handles timestamps across multiple lines`,
		strings.Join([]string{
			"hello\x1b_bk;t=123\x07 world\x1b_bk;t=234\x07!",
			"another\x1b_bk;t=345\x07 line\x1b_bk;t=456\x07!",
		}, "\n"),
		strings.Join([]string{
			`<?bk t="234"?>hello world!`,
			`<?bk t="456"?>another line!`,
		}, "\n"),
	},
}

func TestRendererAgainstCases(t *testing.T) {
	for _, c := range rendererTestCases {
		t.Run(c.name, func(t *testing.T) {
			output := string(Render([]byte(c.input)))
			if output != c.expected {
				t.Errorf("%s\ninput\t\t%q\nexpected\t%q\nreceived\t%q", c.name, c.input, c.expected, output)
			}
		})
	}
}

func TestRendererAgainstFixtures(t *testing.T) {
	for _, base := range TestFiles {
		t.Run(fmt.Sprintf("for fixture %q", base), func(t *testing.T) {
			raw := loadFixture(t, base, "raw")
			expected := string(loadFixture(t, base, "rendered"))

			output := string(Render(raw))

			if output != expected {
				t.Errorf("%s did not match, got len %d and expected len %d", base, len(output), len(expected))
			}
		})
	}
}

func TestScreenWriteToXY(t *testing.T) {
	s := screen{style: &emptyStyle}
	s.write('a')

	s.x = 1
	s.y = 1
	s.write('b')

	s.x = 2
	s.y = 2
	s.write('c')

	output := string(s.asHTML())
	expected := "a\n b\n  c"
	if output != expected {
		t.Errorf("got %q, wanted %q", output, expected)
	}
}

func BenchmarkRendererControl(b *testing.B) {
	benchmark("control.sh", b)
}

func BenchmarkRendererCurl(b *testing.B) {
	benchmark("curl.sh", b)
}

func BenchmarkRendererHomer(b *testing.B) {
	benchmark("homer.sh", b)
}

func BenchmarkRendererDockerPull(b *testing.B) {
	benchmark("docker-pull.sh", b)
}

func BenchmarkRendererPikachu(b *testing.B) {
	benchmark("pikachu.sh", b)
}

func BenchmarkRendererNpm(b *testing.B) {
	benchmark("npm.sh", b)
}

func benchmark(filename string, b *testing.B) {
	raw := loadFixture(b, filename, "raw")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Render(raw)
	}
}

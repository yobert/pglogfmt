package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/yobert/pglogfmt/util"
)

var (
	durationRe = regexp.MustCompile(`\s*duration:\s+[\d\.]+\s+\w+\s+`)
	executeRe  = regexp.MustCompile(`\s*execute\s+[\w\<\>]+:\s+`)
	identRe    = regexp.MustCompile(`"[^"]*"`)
	paramRe    = regexp.MustCompile(`\$\d+`)
	paramSet   = regexp.MustCompile(`(\$\d+)\s*=\s*('[^']*')`)

	simpleMessages = map[string]bool{
		"BEGIN":    true,
		"COMMIT":   true,
		"ROLLBACK": true,
	}
	parseMessages = map[string]bool{
		"SELECT": true,
		"INSERT": true,
		"UPDATE": true,
		"DELETE": true,
	}
)

func main() {
	r := csv.NewReader(os.Stdin)
	ctx := context.Background()

	last := time.Now()

	for {
		row, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		op := row[7]
		txt := row[13]
		params := row[14]

		if !simpleMessages[op] && !parseMessages[op] {
			continue
		}

		t := time.Now()
		if t.Sub(last) > time.Second {
			fmt.Println("────────────────────────────────────────────────────────────")
			last = t
		}

		c := util.ColorSQL{}

		if simpleMessages[op] {
			data := map[string]interface{}{
				"sql": op,
			}
			c.Log(ctx, 0, "", data)
		} else if parseMessages[op] {
			txt = durationRe.ReplaceAllString(txt, "")
			txt = executeRe.ReplaceAllString(txt, "")
			txt = identRe.ReplaceAllStringFunc(txt, func(v string) string {
				v = v[1 : len(v)-1]
				return v
			})

			args := []interface{}{}
			data := map[string]interface{}{
				"sql":  txt,
				"args": args,
			}

			for _, m := range paramSet.FindAllStringSubmatch(params, -1) {
				k := m[1]
				v := m[2]
				d, err := strconv.ParseInt(k[1:], 10, 32)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
				di := int(d) - 1
				if di >= len(args) {
					newargs := make([]interface{}, di+1)
					copy(newargs, args)
					args = newargs
					data["args"] = args
				}
				args[di] = v[1 : len(v)-1]
			}

			c.Log(ctx, 0, "", data)

			/*			paramvals := map[string]string{}
						for _, m := range paramSet.FindAllStringSubmatch(params, -1) {
							paramvals[m[1]] = m[2]
						}
						txt = paramRe.ReplaceAllStringFunc(txt, func(v string) string {
							vv, ok := paramvals[v]
							if ok {
								return vv
							}
							return v
						})
						fmt.Println(txt)*/
		}
	}
}

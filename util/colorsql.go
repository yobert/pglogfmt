package util

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
)

var placeholderRegexp = regexp.MustCompile(`\$\d+`)
var commentRegexp = regexp.MustCompile(`--[^\n]*`)
var spaceRegexp = regexp.MustCompile(`[ \t]+`)
var shorten = true
var null = "NULL"

type ColorSQL struct {
}

func (_ ColorSQL) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	//	fmt.Println(level, msg, data)

	if data == nil {
		return
	}
	s, ok := data["sql"]
	if !ok || s == nil {
		return
	}
	sql, ok := s.(string)
	if !ok {
		return
	}

	color := Blue
	sql = strings.TrimSpace(sql)
	sqlow := strings.ToLower(sql)
	if strings.HasPrefix(sqlow, "update") {
		color = Magenta
	} else if strings.HasPrefix(sqlow, "insert") {
		color = Green
	} else if strings.HasPrefix(sqlow, "delete") {
		color = Red
	} else if strings.HasPrefix(sqlow, "commit") {
		color = Cyan
	} else if strings.HasPrefix(sqlow, "rollback") {
		color = Red
	} else if strings.HasPrefix(sqlow, "begin") {
		color = Cyan
	}

	var args []interface{}
	if data["args"] != nil {
		args = data["args"].([]interface{})

		// trim off QuerySimpleProtocol crap
		if len(args) > 0 {
			_, crap := args[0].(pgx.QuerySimpleProtocol)
			if crap {
				args = args[1:]
			}
		}
	}

	if shorten {
		sql = commentRegexp.ReplaceAllString(sql, "")
		sql = strings.ReplaceAll(sql, "\n", " ")
		sql = spaceRegexp.ReplaceAllString(sql, " ")
	}

	sql = placeholderRegexp.ReplaceAllStringFunc(sql, func(match string) string {
		i, err := strconv.Atoi(match[1:])
		if err != nil {
			return match
		}
		if i < 1 || i > len(args) {
			return match
		}
		v := args[i-1]
		var nice string
		switch vv := v.(type) {
		case string:
			if len(vv) > 64 {
				vv = vv[:64] + fmt.Sprintf(" (truncated %d bytes)", len(vv)-64)
			}
			nice = quote(vv)
		case *string:
			if vv == nil {
				nice = null
			} else {
				if len(*vv) > 64 {
					s := *vv
					nice = quote(s[:64] + fmt.Sprintf(" (truncated %d bytes)", len(s)-64))
				} else {
					nice = quote(*vv)
				}
			}
		case int:
			nice = fmt.Sprintf("%d", vv)
		case *int:
			if vv == nil {
				nice = null
			} else {
				nice = fmt.Sprintf("%d", *vv)
			}
		case bool:
			nice = strconv.FormatBool(vv)
		case *bool:
			if vv == nil {
				nice = null
			} else {
				nice = strconv.FormatBool(*vv)
			}
		case time.Time:
			nice = vv.String()
		case *time.Time:
			if vv == nil {
				nice = null
			} else {
				nice = vv.String()
			}
		default:
			fmt.Fprintf(os.Stdout, "what is %#v (%T)?\n", v, v)
			return match
		}
		return BrightText(color) + nice + Reset() + Text(color)
	})
	fmt.Fprintf(os.Stdout, "%s%s%s\n", Text(color), sql, Reset())
}

func quote(v string) string {
	r := make([]byte, 0, len(v)+3)
	e := false

	r = append(r, 'E', '\'')

	for i := 0; i < len(v); i++ {
		ch := v[i]
		if ch == '\r' {
			e = true
			r = append(r, '\\', 'r')
		} else if ch == '\n' {
			e = true
			r = append(r, '\\', 'n')
		} else if ch < 32 || ch > 126 || ch == '\\' || ch == '\'' {
			e = true
			hx := fmt.Sprintf("%02x", ch)
			r = append(r, '\\', 'x', hx[0], hx[1])
		} else {
			r = append(r, ch)
		}
	}

	r = append(r, '\'')

	if e {
		return string(r)
	} else {
		return string(r[1:])
	}
}

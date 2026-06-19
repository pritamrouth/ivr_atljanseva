package plivo

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

func Response(children ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("<Response>")
	for _, c := range children {
		b.WriteString(c)
	}
	b.WriteString("</Response>")
	return b.String()
}

func Speak(text, lang string) string {
	l := languageCode(lang)
	return fmt.Sprintf(`<Speak language="%s">%s</Speak>`, l, escape(text))
}

func Play(url string) string {
	return fmt.Sprintf(`<Play>%s</Play>`, escape(url))
}

func GetDigits(action string, numDigits, timeout int, children ...string) string {
	inner := strings.Join(children, "")
	return fmt.Sprintf(
		`<GetDigits action="%s" method="POST" numDigits="%d" timeout="%d">%s</GetDigits>`,
		escape(action), numDigits, timeout, inner,
	)
}

func GetDigitsEx(action string, numDigits, timeout int, finishOnKey string, digitTimeout int, children ...string) string {
	inner := strings.Join(children, "")
	return fmt.Sprintf(
		`<GetDigits action="%s" method="POST" numDigits="%d" timeout="%d" finishOnKey="%s" digitTimeout="%d">%s</GetDigits>`,
		escape(action), numDigits, timeout, escape(finishOnKey), digitTimeout, inner,
	)
}

func Hangup() string {
	return `<Hangup/>`
}

func Redirect(url string) string {
	return fmt.Sprintf(`<Redirect>%s</Redirect>`, escape(url))
}

func Dial(number string) string {
	return fmt.Sprintf(`<Dial callerId="%s">%s</Dial>`, escape(number), escape(number))
}

func DialWithAction(number, actionURL string) string {
	return fmt.Sprintf(
		`<Dial action="%s" method="POST" callerId="%s">%s</Dial>`,
		escape(actionURL), escape(number), escape(number),
	)
}

func Record(action string, maxSeconds int, finishOnKey string) string {
	return fmt.Sprintf(
		`<Record action="%s" method="POST" maxLength="%d" finishOnKey="%s" />`,
		escape(action), maxSeconds, escape(finishOnKey),
	)
}

func RecordWithBeep(action string, maxSeconds int, finishOnKey string) string {
	return fmt.Sprintf(
		`<Record action="%s" method="POST" maxLength="%d" finishOnKey="%s" playBeep="true" />`,
		escape(action), maxSeconds, escape(finishOnKey),
	)
}

func Wait(seconds int) string {
	return fmt.Sprintf(`<Wait length="%d"/>`, seconds)
}

func AudioURL(baseURL, lang, file string) string {
	encoded := url.PathEscape(file)
	return strings.TrimRight(baseURL, "/") + "/" + langCodeShort(lang) + "/" + encoded
}

func languageCode(lang string) string {
	switch strings.ToLower(lang) {
	case "english":
		return "en-IN"
	case "hindi":
		return "hi-IN"
	case "marathi":
		return "mr-IN"
	default:
		return "en-IN"
	}
}

func langCodeShort(lang string) string {
	switch strings.ToLower(lang) {
	case "english":
		return "audio-eng"
	case "hindi":
		return "audio-hindi"
	case "marathi":
		return "audio-mr"
	default:
		return "audio-eng"
	}
}

func escape(s string) string {
	return html.EscapeString(s)
}

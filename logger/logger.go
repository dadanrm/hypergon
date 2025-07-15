package logger

import (
	"fmt"
	"log"
	"strings"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

func INFO(args ...interface{}) {
	var sb strings.Builder

	for i, arg := range args {
		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(fmt.Sprint(arg))
	}
	log.Println(ColorCyan + "INFO: " + sb.String() + ColorReset)
}

func ERROR(args ...interface{}) {
	var sb strings.Builder

	for i, arg := range args {
		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(fmt.Sprint(arg))
	}

	log.Println(ColorRed + "ERROR: " + sb.String() + ColorReset)
}

func WARN(args ...interface{}) {
	var sb strings.Builder

	for i, arg := range args {
		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(fmt.Sprint(arg))
	}
	log.Println(ColorYellow + "WARNING: " + sb.String() + ColorReset)
}

func LOG(args ...interface{}) {
	var sb strings.Builder

	for i, arg := range args {
		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(fmt.Sprint(arg))
	}
	log.Println(ColorGreen + sb.String() + ColorReset)
}

func BAD(args ...interface{}) {
	var sb strings.Builder

	for i, arg := range args {
		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(fmt.Sprint(arg))
	}
	log.Println(ColorPurple + "BAD: " + sb.String() + ColorReset)
}

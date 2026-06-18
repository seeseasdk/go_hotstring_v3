package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	followUpRegex := regexp.MustCompile(`f(6m|\d+|f|c|a|o)`)

	tests := []string{"ff", "fa", "fc", "fo", "f6m", "f3", "lm4bff", "lm4bfa", "lm4beff", "lm4befa", "fib", "ftb"}
	for _, buf := range tests {
		fuMatches := followUpRegex.FindAllStringSubmatchIndex(buf, -1)
		fmt.Printf("Buffer: %-15s -> FollowUp matches: %v\n", buf, fuMatches)
	}

	fmt.Println()

	// Test K_Blocks key lookup for "ff" and "fa"
	kBlockKeys := []string{
		"cm3b", "cm3r", "cm3l", "cf3b", "cr5b", "lm4b", "lm4r", "lm4l",
		"fib", "fir", "fil", "ftb", "ftr", "ftl",
		"pfb", "pfr", "pfl", "pfib", "pfir", "pfil",
		"pftb", "pftr", "pftl",
		"shb", "shr", "shl", "knb", "knr", "knl", "akb", "akr", "akl",
		"wrb", "wrr", "wrl", "ebb", "ebr", "ebl",
	}

	for _, buf := range []string{"ff", "fa", "fc", "fo"} {
		fmt.Printf("Buffer '%s':\n", buf)
		for _, k := range kBlockKeys {
			if idx := strings.Index(buf, k); idx != -1 {
				fmt.Printf("  K_Blocks key '%s' found at position %d!\n", k, idx)
			}
			// c-prefix
			target_c := "c" + k
			if idx := strings.Index(buf, target_c); idx != -1 {
				fmt.Printf("  K_Blocks c-prefix 'c%s' found at position %d!\n", k, idx)
			}
			// p-prefix
			target_p := "p" + k
			if idx := strings.Index(buf, target_p); idx != -1 {
				fmt.Printf("  K_Blocks p-prefix 'p%s' found at position %d!\n", k, idx)
			}
		}
		fmt.Println("  (no other matches)")
	}
}

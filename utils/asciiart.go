package utils

import "fmt"

func PrintASCII(port string, rootDir string, version string) {
	if rootDir == "" {
		rootDir = "<unspecified>"
	}
	fmt.Println(" ________  ___  ________      			")
	fmt.Println("|\\_____  \\|\\  \\|\\   ___  \\    	")
	fmt.Println(" \\|___/  /\\ \\  \\ \\  \\\\ \\  \\   ")
	fmt.Println("     /  / /\\ \\  \\ \\  \\\\ \\  \\  	")
	fmt.Println("    /  /_/__\\ \\  \\ \\  \\\\ \\  \\ 	")
	fmt.Println("   |\\________\\ \\__\\ \\__\\\\ \\__\\")
	fmt.Println("    \\|_______|\\|__|\\|__| \\|__|		")
	fmt.Printf("\n• Version : %s", version)
	fmt.Printf("\n• Source  : %s", rootDir)
	fmt.Printf("\n• Local   : http://127.0.0.1:%s\n", port)
}

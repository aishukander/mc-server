package main

import (
	"fmt"
	"os"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("get current working directory failed:", err)
		os.Exit(1)
	}
	java_path := fmt.Sprintf("%s/java", cwd)
	server_path := fmt.Sprintf("%s/server", cwd)

	java_version := Get_java_version()
	fmt.Printf("Determined Java version: %s\n", java_version)

	if err := Install_Java_from_git(java_version, java_path); err != nil {
		fmt.Printf("GitHub installation failed: %s\n", err)
		os.Exit(1)
	}

	if err := Create_eula(server_path); err != nil {
		fmt.Printf("Failed to create eula.txt: %s\n", err)
		os.Exit(1)
	}

	if err := os.Chmod(fmt.Sprintf("%s/jdk%s/bin/java", java_path, java_version), 0755); err != nil {
		fmt.Printf("Failed to set executable permissions on Java: %s\n", err)
		os.Exit(1)
	}

	java_bin_path := fmt.Sprintf("%s/jdk%s/bin", java_path, java_version)
	server_type := os.Getenv("Type")
	switch server_type {
	case "paper":
		if err := Handle_paper(server_path, java_bin_path); err != nil {
			fmt.Printf("Failed to handle PaperMC: %s\n", err)
			os.Exit(1)
		}
	case "neoforge":
		if err := Handle_neoforge(server_path, java_bin_path); err != nil {
			fmt.Printf("Failed to handle NeoForge: %s\n", err)
			os.Exit(1)
		}
	default:
		if err := Handle_other(server_path, server_type, java_bin_path); err != nil {
			fmt.Printf("Failed to handle other server types: %s\n", err)
			os.Exit(1)
		}
	}
}

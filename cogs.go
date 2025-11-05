package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func Version_greater(v1, v2 string) bool {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := max(len(parts2), len(parts1))

	for i := range maxLen {
		var p1, p2 int
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return true
		}
		if p1 < p2 {
			return false
		}
	}

	return true
}

func Get_java_version() string {
	if overrideVersion := os.Getenv("JAVA_VERSION_OVERRIDE"); overrideVersion != "" {
		return overrideVersion
	}

	minecraftVersion := os.Getenv("MINECRAFT_VERSION")
	if minecraftVersion == "" {
		return "8"
	}

	switch {
	case Version_greater(minecraftVersion, "1.20.5"):
		return "21"
	case Version_greater(minecraftVersion, "1.18"):
		return "17"
	case Version_greater(minecraftVersion, "1.17"):
		return "16"
	default:
		return "8"
	}
}

func Install_Java_from_git(version string, path string) error {
	installPath := fmt.Sprintf("%s/jdk%s", path, version)

	info, err := os.Stat(installPath)
	if err == nil && info.IsDir() {
		fmt.Printf("Java version %s is already installed at %s\n", version, installPath)
		return nil
	}

	fmt.Printf("Starting installation of Java %s\n", version)

	apiURL := fmt.Sprintf("https://api.github.com/repos/adoptium/temurin%s-binaries/releases/latest", version)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch release info from GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned non-200 status: %s", resp.Status)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse JSON response from GitHub API: %w", err)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, "jdk_x64_linux") && strings.HasSuffix(asset.Name, ".tar.gz") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("could not find a download URL for Java %s for x64 Linux", version)
	}

	fmt.Printf("Downloading from %s\n", downloadURL)

	archiveName := fmt.Sprintf("jdk%s.tar.gz", version)
	out, err := os.Create(archiveName)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer out.Close()

	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Java archive: %w", err)
	}
	defer downloadResp.Body.Close()

	if _, err := io.Copy(out, downloadResp.Body); err != nil {
		return fmt.Errorf("failed to save Java archive: %w", err)
	}

	file, err := os.Open(archiveName)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var rootDir string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if rootDir == "" {
			rootDir = strings.Split(header.Name, "/")[0]
		}

		target := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory during extraction: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file during extraction: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file content during extraction: %w", err)
			}
			outFile.Close()
		}
	}

	if rootDir == "" {
		return fmt.Errorf("could not determine root directory from archive")
	}

	fmt.Printf("Moving %s to %s\n", rootDir, installPath)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", path, err)
	}

	cp_cmd := exec.Command("cp", "-r", rootDir, installPath)
	if err := cp_cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy extracted directory: %w", err)
	}
	if err := os.RemoveAll(rootDir); err != nil {
		return fmt.Errorf("failed to remove original extracted directory: %w", err)
	}

	if err := os.Remove(archiveName); err != nil {
		fmt.Printf("Warning: failed to remove archive %s: %v\n", archiveName, err)
	}

	fmt.Println("Java installation completed successfully.")
	return nil
}

func Create_eula(path string) error {
	if _, err := os.Stat(fmt.Sprintf("%s/eula.txt", path)); os.IsNotExist(err) {
		fmt.Println("eula.txt not found, creating it.")

		timestamp := time.Now().UTC().Format("# Mon Jan 02 03:04:05 PM MST 2006")
		content := fmt.Sprintf(`
			# Created with Docker
			%s
			eula=true
			`, timestamp)

		if err := os.WriteFile(fmt.Sprintf("%s/eula.txt", path), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write to eula.txt: %w", err)
		}

		fmt.Println("eula.txt created successfully.")
	} else if err != nil {
		return fmt.Errorf("error checking eula.txt: %w", err)
	} else {
		fmt.Println("eula.txt already exists.")
	}

	return nil
}

func Set_java_executable(java_bin_path string) error {
	entries, err := os.ReadDir(java_bin_path)
	if err != nil {
		return fmt.Errorf("failed to read java bin directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := fmt.Sprintf("%s/%s", java_bin_path, entry.Name())
			if err := os.Chmod(filePath, 0755); err != nil {
				return fmt.Errorf("failed to set executable permission for %s: %w", filePath, err)
			}
		}
	}

	fmt.Printf("Set executable permissions for Java binaries in %s\n", java_bin_path)
	return nil
}

func Handle_paper(server_path string, java_bin_path string) error {
	minecraft_version := os.Getenv("MINECRAFT_VERSION")
	jar_name := fmt.Sprintf("paper-%s.jar", minecraft_version)
	jar_path := fmt.Sprintf("%s/%s", server_path, jar_name)

	if _, err := os.Stat(jar_path); os.IsNotExist(err) {
		fmt.Printf("%s not found, downloading...\n", jar_name)
		api_url := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds", minecraft_version)

		resp, err := http.Get(api_url)
		if err != nil {
			return fmt.Errorf("failed to fetch builds from PaperMC API: %w", err)
		}
		defer resp.Body.Close()

		var builds struct {
			Builds []struct {
				Build int `json:"build"`
			} `json:"builds"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
			return fmt.Errorf("failed to parse JSON from PaperMC API: %w", err)
		}

		if len(builds.Builds) == 0 {
			return fmt.Errorf("unsupported version of Minecraft: %s", minecraft_version)
		}
		latest_build := builds.Builds[len(builds.Builds)-1].Build

		download_url := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds/%d/downloads/paper-%s-%d.jar", minecraft_version, latest_build, minecraft_version, latest_build)

		out, err := os.Create(jar_path)
		if err != nil {
			return fmt.Errorf("failed to create paper jar file: %w", err)
		}
		defer out.Close()

		downloadResp, err := http.Get(download_url)
		if err != nil {
			return fmt.Errorf("failed to download paper jar: %w", err)
		}
		defer downloadResp.Body.Close()

		if downloadResp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status while downloading paper jar: %s", downloadResp.Status)
		}

		_, err = io.Copy(out, downloadResp.Body)
		if err != nil {
			return fmt.Errorf("failed to save paper jar: %w", err)
		}
	}

	min_ram := os.Getenv("Min_Ram")
	max_ram := os.Getenv("Max_Ram")
	paper_start := exec.Command("sh", "-c", fmt.Sprintf("export PATH=%s:$PATH && java -Xms%s -Xmx%s -jar %s nogui", java_bin_path, min_ram, max_ram, jar_name))
	paper_start.Dir = server_path
	paper_start.Stdout = os.Stdout
	paper_start.Stdin = os.Stdin
	paper_start.Stderr = os.Stderr
	return paper_start.Run()
}

func Handle_neoforge(server_path string, java_bin_path string) error {
	min_ram := os.Getenv("Min_Ram")
	max_ram := os.Getenv("Max_Ram")
	jvm_args := fmt.Sprintf("-Xms%s -Xmx%s", min_ram, max_ram)
	if err := os.WriteFile(fmt.Sprintf("%s/user_jvm_args.txt", server_path), []byte(jvm_args), 0644); err != nil {
		return fmt.Errorf("failed to write user_jvm_args.txt: %w", err)
	}

	neo_version_override := os.Getenv("NEO_VERSION_OVERRIDE")
	if neo_version_override != "" {
		if _, err := os.Stat(fmt.Sprintf("%s/libraries/net/neoforged/neoforge/%s", server_path, neo_version_override)); os.IsNotExist(err) {
			fmt.Println("Changing the neoforge version...")
			os.RemoveAll(fmt.Sprintf("%s/libraries", server_path))
			os.RemoveAll(fmt.Sprintf("%s/logs", server_path))
			os.Remove(fmt.Sprintf("%s/run.sh", server_path))
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/run.sh", server_path)); os.IsNotExist(err) {
		minecraft_version := os.Getenv("MINECRAFT_VERSION")
		version_filter := strings.TrimPrefix(minecraft_version, "1.")
		api_url := "https://maven.neoforged.net/releases/net/neoforged/neoforge"

		neo_version := neo_version_override
		if neo_version == "" {
			metadata_url := fmt.Sprintf("%s/maven-metadata.xml", api_url)
			// This is a simplified way to get the latest version without a full XML parser
			// It might break if the XML structure changes.
			resp, err := http.Get(metadata_url)
			if err != nil {
				return fmt.Errorf("failed to get neoforge metadata: %w", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read neoforge metadata: %w", err)
			}

			versions := []string{}
			lines := strings.Split(string(body), "\n")
			for _, line := range lines {
				if strings.Contains(line, fmt.Sprintf("<version>%s", version_filter)) {
					v := strings.TrimSpace(line)
					v = strings.TrimPrefix(v, "<version>")
					v = strings.TrimSuffix(v, "</version>")
					versions = append(versions, v)
				}
			}
			if len(versions) == 0 {
				return fmt.Errorf("unsupported version of Minecraft for Neoforge: %s", minecraft_version)
			}
			neo_version = versions[len(versions)-1]
		}

		installer_name := fmt.Sprintf("neoforge-%s-installer.jar", neo_version)
		installer_url := fmt.Sprintf("%s/%s/%s", api_url, neo_version, installer_name)
		installer_path := fmt.Sprintf("%s/neoforge-installer.jar", server_path)

		out, err := os.Create(installer_path)
		if err != nil {
			return fmt.Errorf("failed to create neoforge installer file: %w", err)
		}
		defer out.Close()

		resp, err := http.Get(installer_url)
		if err != nil {
			return fmt.Errorf("failed to download neoforge installer: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status while downloading neoforge installer: %s", resp.Status)
		}

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save neoforge installer: %w", err)
		}

		neoforge_installer := exec.Command("sh", "-c", fmt.Sprintf("export PATH=%s:$PATH && java -jar neoforge-installer.jar", java_bin_path))
		neoforge_installer.Dir = server_path
		neoforge_installer.Stdout = os.Stdout
		neoforge_installer.Stderr = os.Stderr
		if err := neoforge_installer.Run(); err != nil {
			return fmt.Errorf("failed to run neoforge installer: %w", err)
		}
		os.Remove(installer_path)
	}

	neoforge_start := exec.Command("sh", "-c", fmt.Sprintf("export PATH=%s:$PATH && ./run.sh", java_bin_path))
	neoforge_start.Dir = server_path
	neoforge_start.Stdout = os.Stdout
	neoforge_start.Stdin = os.Stdin
	neoforge_start.Stderr = os.Stderr
	return neoforge_start.Run()
}

func Handle_other(server_path, server_type string, java_bin_path string) error {
	minecraft_version := os.Getenv("MINECRAFT_VERSION")
	jar_name := fmt.Sprintf("%s-%s.jar", server_type, minecraft_version)
	jar_path := fmt.Sprintf("%s/%s", server_path, jar_name)

	if _, err := os.Stat(jar_path); os.IsNotExist(err) {
		fmt.Printf("%s not found, downloading...\n", jar_name)
		download_url := fmt.Sprintf("https://mcutils.com/api/server-jars/%s/%s/download", server_type, minecraft_version)

		out, err := os.Create(jar_path)
		if err != nil {
			return fmt.Errorf("failed to create server jar file: %w", err)
		}
		defer out.Close()

		resp, err := http.Get(download_url)
		if err != nil {
			return fmt.Errorf("failed to download server jar: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status while downloading server jar: %s", resp.Status)
		}

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save server jar: %w", err)
		}
	}

	min_ram := os.Getenv("Min_Ram")
	max_ram := os.Getenv("Max_Ram")
	other_start := exec.Command("sh", "-c", fmt.Sprintf("export PATH=%s:$PATH && java -Xms%s -Xmx%s -jar %s nogui", java_bin_path, min_ram, max_ram, jar_name))
	other_start.Dir = server_path
	other_start.Stdout = os.Stdout
	other_start.Stdin = os.Stdin
	other_start.Stderr = os.Stderr
	return other_start.Run()
}

package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install SteamCMD",
	Long: `Install SteamCMD to the configured directory.

This command will download and extract SteamCMD based on your operating system:
- Windows: Downloads steamcmd.zip
- Linux: Downloads steamcmd_linux.tar.gz  
- macOS: Downloads steamcmd_osx.tar.gz

The SteamCMD will be installed to the directory specified in configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return installSteamCMD()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolP("force", "f", false, "Force reinstall even if SteamCMD already exists")
	viper.BindPFlag("force_install", installCmd.Flags().Lookup("force"))
}

func installSteamCMD() error {
	steamcmdDir := viper.GetString("steamcmd_dir")
	force := viper.GetBool("force_install")

	// Check if SteamCMD already exists
	var steamcmdExe string
	if runtime.GOOS == "windows" {
		steamcmdExe = filepath.Join(steamcmdDir, "steamcmd.exe")
	} else {
		steamcmdExe = filepath.Join(steamcmdDir, "steamcmd.sh")
	}

	if !force {
		if _, err := os.Stat(steamcmdExe); err == nil {
			fmt.Printf("SteamCMD already exists at %s\n", steamcmdExe)
			fmt.Println("Use --force to reinstall")
			return nil
		}
	}

	// Create steamcmd directory
	if err := os.MkdirAll(steamcmdDir, 0755); err != nil {
		return fmt.Errorf("failed to create SteamCMD directory: %w", err)
	}

	// Get download URL based on OS
	downloadURL, filename := getSteamCMDDownloadURL()

	fmt.Printf("Downloading SteamCMD from %s...\n", downloadURL)

	// Download SteamCMD
	tempFile := filepath.Join(steamcmdDir, filename)
	if err := downloadFile(downloadURL, tempFile); err != nil {
		return fmt.Errorf("failed to download SteamCMD: %w", err)
	}

	fmt.Println("Extracting SteamCMD...")

	// Extract based on file type
	if err := extractSteamCMD(tempFile, steamcmdDir); err != nil {
		return fmt.Errorf("failed to extract SteamCMD: %w", err)
	}

	// Remove temporary file
	os.Remove(tempFile)

	fmt.Printf("SteamCMD successfully installed to %s\n", steamcmdDir)

	// Run initial SteamCMD update
	fmt.Println("Running initial SteamCMD update...")
	if err := runInitialSteamCMDUpdate(steamcmdExe); err != nil {
		fmt.Printf("Warning: Initial update failed: %v\n", err)
		fmt.Println("You may need to run SteamCMD manually the first time")
	} else {
		fmt.Println("SteamCMD installation completed successfully!")
	}

	return nil
}

func getSteamCMDDownloadURL() (string, string) {
	baseURL := "https://steamcdn-a.akamaihd.net/client/installer/"

	switch runtime.GOOS {
	case "windows":
		return baseURL + "steamcmd.zip", "steamcmd.zip"
	case "darwin":
		return baseURL + "steamcmd_osx.tar.gz", "steamcmd_osx.tar.gz"
	default: // linux
		return baseURL + "steamcmd_linux.tar.gz", "steamcmd_linux.tar.gz"
	}
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractSteamCMD(archivePath, destDir string) error {
	if runtime.GOOS == "windows" {
		return extractZip(archivePath, destDir)
	} else {
		return extractTarGz(archivePath, destDir)
	}
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create destination directory if it doesn't exist
	os.MkdirAll(dest, 0755)

	// Extract files and folders
	for _, f := range r.File {
		// Create the destination path
		path := filepath.Join(dest, f.Name)

		// Security check: ensure the file path is within the destination directory
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// Create directory
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		// Create the directory for the file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		// Ensure the target is within dest directory
		if !filepath.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

func runInitialSteamCMDUpdate(steamcmdPath string) error {
	// Run steamcmd +quit to trigger the initial update
	cmd := exec.Command(steamcmdPath, "+quit")
	cmd.Dir = filepath.Dir(steamcmdPath)

	// Capture output for verbose mode
	var outputBuf strings.Builder
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	if err != nil {
		if viper.GetBool("verbose") {
			fmt.Printf("SteamCMD output:\n%s\n", outputBuf.String())
		}
		return fmt.Errorf("initial update failed: %w", err)
	}

	// Check if the output indicates successful update
	output := outputBuf.String()
	if strings.Contains(output, "Loading Steam API...OK") {
		fmt.Println("Initial SteamCMD update completed successfully")
		return nil
	}

	if viper.GetBool("verbose") {
		fmt.Printf("SteamCMD output:\n%s\n", output)
	}

	return nil
}

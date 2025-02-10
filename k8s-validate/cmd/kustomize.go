package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
)

func findFolders() []string {
	// Find all directories containing kustomization.yaml
	var folders []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If the current file is kustomization.yaml, add its directory to the list
		if info.Name() == "kustomization.yaml" {
			dir := filepath.Dir(path)
			folders = append(folders, dir)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error finding kustomize folders:", err)
		os.Exit(1)
	}

	// Sort the folders
	sort.Strings(folders)

	return folders
}

func validate(folder string, failFast bool) bool {
	var valid = true

	// Build the kustomization.yaml with kustomize
	cmd := exec.Command("kustomize", "build", folder)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running kustomize build on %s: %v\n", folder, err)
		valid = false
	}
	if valid {
		fmt.Printf("Running kustomize build on: %-50s OK\n", folder+"...")
	} else {
		fmt.Printf("Running kustomize build on: %-50s ERROR\n", folder+"...")
		if failFast {
			os.Exit(1)
		}
	}
	return valid
}

func ValidateKustomize(workers int, failFast bool) bool {
	var folders = findFolders()

	if len(folders) == 0 {
		fmt.Println("No kustomize folder found")
		return true
	}

	var wg sync.WaitGroup
	taskChan := make(chan string, len(folders)) // Buffered channel for tasks

	// Start worker goroutines
	var valid = true
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for folder := range taskChan { // Process tasks from channel
				if !validate(folder, failFast) {
					valid = false
				}
			}
		}()
	}

	// Send tasks to workers
	for _, folder := range folders {
		taskChan <- folder
	}
	close(taskChan) // Close channel to signal workers no more tasks are coming

	// Wait for all workers to finish
	wg.Wait()

	return valid
}

package main

import (
	"fmt"
	"log"
)

func util() {

	localFolderPath := "./files_to_deploy"
	zipFilePath := "./files_to_deploy.zip"
	remoteZipPath := fmt.Sprintf("/home/%s/uploaded/files_to_deploy.zip", username)
	remoteFolderPath := fmt.Sprintf("/home/%s/uploaded/files_to_deploy", username)

	err = compressFolder(localFolderPath, zipFilePath)
	if err != nil {
		log.Fatalf("Failed to compress folder: %v", err)
	}

	for _, instance := range instances {
		client, err := connect(instance, username, ssh_key)
		if err != nil {
			fmt.Printf("Error connecting to instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		defer client.Close()
		output := ""
		output, err = runCommand(client, "sudo apt install unzip")
		if err != nil {
			fmt.Printf("Error running command on instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		fmt.Printf("Output from instance %s: %s\n", instance.ExternalIP, output)

		output, err = runCommand(client, "rm -rf ~/uploaded && mkdir -p ~/uploaded")
		if err != nil {
			fmt.Printf("Error running command on instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		fmt.Printf("Output from instance %s: %s\n", instance.ExternalIP, output)

		output, err = runCommand(client, "ls -la ~/ && tree -d ~/uploaded")
		if err != nil {
			fmt.Printf("Error running command on instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		fmt.Printf("Output from instance %s: %s\n", instance.ExternalIP, output)

		err = uploadFile(client, zipFilePath, remoteZipPath)
		if err != nil {
			fmt.Printf("Error uploading file to instance %s: %v\n", instance.ExternalIP, err)
			continue
		}

		output, err = runCommand(client, fmt.Sprintf("unzip -o %s -d %s", remoteZipPath, remoteFolderPath))
		if err != nil {
			fmt.Printf("Error running command on instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		fmt.Printf("Output from instance %s: %s\n", instance.ExternalIP, output)

		output, err = runCommand(client, fmt.Sprintf("tree -d %s", remoteFolderPath))
		if err != nil {
			fmt.Printf("Error running command on instance %s: %v\n", instance.ExternalIP, err)
			continue
		}
		fmt.Printf("Output from instance %s: %s\n", instance.ExternalIP, output)
	}
}

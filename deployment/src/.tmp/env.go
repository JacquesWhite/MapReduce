package main

// import (
// 	"bytes"
// 	"fmt"
// 	"os"
// 	"os/exec"
// 	"strings"

// 	"github.com/joho/godotenv"
// )

// type Env struct {
// 	projectId         string
// 	zone              string
// 	instanceGroupName string
// 	username          string
// 	ssh_key           string
// }

// var env Env

// func main() {
// 	loadEnvFile()
// 	setProjectID()
// 	setZoneAndInstanceGroupName()
// 	setUsername()
// 	setSSHKey()
// }

// func loadEnvFile() {
// 	err := godotenv.Load("../.env")
// 	if err != nil {
// 		fmt.Errorf("Error loading .env file: ", err)
// 		fmt.Println("Creating .env file from .env.template")
// 		dotEnvExample, err := os.ReadFile("../.env.template")
// 		if err != nil {
// 			fmt.Errorf("Error reading .env.template file: %v", err)
// 		}
// 		err = os.WriteFile(".env", dotEnvExample, 0644)
// 		if err != nil {
// 			fmt.Errorf("Error creating .env file: %v", err)
// 			os.Exit(1)
// 		}
// 		err = godotenv.Load("../.env")
// 		if err != nil {
// 			fmt.Errorf("Error loading .env file: ", err)
// 			os.Exit(1)
// 		}
// 	}
// }

// func setProjectID() {
// 	env.projectId = os.Getenv("GCP_PROJECT_ID")
// 	if env.projectId == "" {
// 		fmt.Errorf("GCP_PROJECT_ID not set in .env file")
// 		fmt.Println("getting list of projects from cloud projects list")
// 		cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
// 		var out bytes.Buffer
// 		cmd.Stdout = &out
// 		err := cmd.Run()
// 		if err != nil {
// 			fmt.Errorf("failed to execute gcloud command: ", err)
// 			fmt.Println("install gcloud sdk and authenticate with gcloud init")
// 		}
// 		projectIDs := strings.Split(strings.TrimSpace(out.String()), "\n")
// 		fmt.Println("setting GCP_PROJECT_ID in .env file")
// 		env.projectId = projectIDs[0]
// 		updateEnvFile("GCP_PROJECT_ID", env.projectId)
// 	}
// }

// func setZoneAndInstanceGroupName() {
// 	env.zone = os.Getenv("GCP_ZONE")
// 	env.instanceGroupName = os.Getenv("GCP_INSTANCE_GROUP_NAME")
// 	if env.zone == "" || env.instanceGroupName == "" {
// 		fmt.Errorf("GCP_ZONE or GCP_INSTANCE_GROUP_NAME not set in .env file")
// 		fmt.Println("getting list instance groups from cloud compute instance-groups list")
// 		cmd := exec.Command("gcloud", "compute", "instance-groups", "list", "--format=value(name,zone)")
// 		var out bytes.Buffer
// 		cmd.Stdout = &out
// 		err := cmd.Run()
// 		if err != nil {
// 			fmt.Errorf("failed to execute gcloud command: ", err)
// 			fmt.Println("install gcloud sdk and authenticate with gcloud init")
// 		}
// 		instanceGroups := strings.Split(strings.TrimSpace(out.String()), "\n")
// 		env.zone = strings.Split(instanceGroups[0], " ")[1]
// 		env.instanceGroupName = strings.Split(instanceGroups[0], " ")[0]
// 		fmt.Println("setting GCP_ZONE and GCP_INSTANCE_GROUP_NAME in .env file")
// 		updateEnvFile("GCP_ZONE", env.zone)
// 		updateEnvFile("GCP_INSTANCE_GROUP_NAME", env.instanceGroupName)
// 	}
// }

// func setUsername() {
// 	env.username = os.Getenv("SSH_USERNAME")
// 	if env.username == "" {
// 		fmt.Errorf("SSH_USERNAME must be set in .env file")
// 		fmt.Println("setting SSH_USERNAME to $USERNAME in .env file")
// 		env.username = os.Getenv("USERNAME")
// 		updateEnvFile("SSH_USERNAME", env.username)
// 	}
// }

// func setSSHKey() {
// 	env.ssh_key = os.Getenv("SSH_KEY")
// 	if env.ssh_key == "" {
// 		fmt.Errorf("SSH_KEY must be set in .env file")
// 		fmt.Println("running gcloud compute ssh to get set up key")
// 		cmd := exec.Command("gcloud", "compute", "ssh", "--zone", env.zone, env.instanceGroupName, "--command", "echo \"...\"")
// 		err := cmd.Run()
// 		if err != nil {
// 			fmt.Errorf("failed to execute gcloud command: ", err)
// 			fmt.Println("install gcloud sdk and authenticate with gcloud init")
// 		}
// 		env.ssh_key = "/Users/" + os.Getenv("USERNAME") + "/.ssh/google_compute_engine"
// 		updateEnvFile("SSH_KEY", env.ssh_key)
// 	}
// }

// func updateEnvFile(key, value string) {
// 	fileContent, err := os.ReadFile(".env")
// 	if err != nil {
// 		fmt.Errorf("failed to read .env file: ", err)
// 	}
// 	newContent := strings.Replace(string(fileContent), key+"=", fmt.Sprintf("%s=%s", key, value), 1)
// 	err = os.WriteFile(".env", []byte(newContent), 0644)
// 	if err != nil {
// 		fmt.Errorf("failed to write to .env file: ", err)
// 	}
// }

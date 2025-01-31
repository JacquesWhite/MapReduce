package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func publicKeyFile(file string) (ssh.AuthMethod, error) {
	key, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func connect(instance Instance, username string, ssh_key string) (*ssh.Client, error) {
	authMethod, err := publicKeyFile(ssh_key)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", instance.ExternalIP+":22", config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return client, nil
}

func runCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %w", err)
	}

	return string(output), nil
}

func uploadFile(client *ssh.Client, localPath, remotePath string) error {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create sftp client: %w", err)
	}
	defer sftpClient.Close()
	fmt.Println("local path:", localPath)
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()
	fmt.Println("local file:", localFile)

	fmt.Println("remote path:", remotePath)
	remoteFile, err := sftpClient.OpenFile(remotePath, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		return err
	}
	defer remoteFile.Close()
	fmt.Println("remote file:", remoteFile)

	if _, err := localFile.Seek(0, 0); err != nil {
		return err
	}
	if _, err := remoteFile.ReadFrom(localFile); err != nil {
		return err
	}

	return nil
}

func compressFolder(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	return nil
}

package awsutil

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

func NewSession() *session.Session {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	return sess
}

func NewSessionExplicit(accessKeyId, secretAccessKey string) *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKeyId, secretAccessKey, ""),
	})
	if err != nil {
		log.Fatal(err)
	}
	return sess
}

func ReadAccessKeyFromStdin() (string, string) {
	fmt.Print("AWS access key ID: ")
	stdin := bufio.NewReader(os.Stdin)
	accessKeyId, err := stdin.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("AWS secret access key: ")
	secretAccessKey, err := stdin.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSuffix(accessKeyId, "\n"), strings.TrimSuffix(secretAccessKey, "\n")
}

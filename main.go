package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/joho/godotenv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"

	"github.com/labstack/echo/middleware"

	"gopkg.in/go-playground/validator.v9"

	"github.com/labstack/echo"
)

type (
	User struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}

	AWSKinesis struct {
		stream          string
		region          string
		endpoint        string
		accessKeyID     string
		secretAccessKey string
		sessionToken    string
	}
)

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func httpError(code int, err error) *echo.HTTPError {
	// https://github.com/labstack/echo/issues/1079
	return echo.NewHTTPError(code, strings.ToLower(err.Error()))
}

func getUser(c echo.Context) (err error) {
	u := new(User)
	if err = c.Bind(u); err != nil {
		return
	}
	if err = c.Validate(u); err != nil {
		return httpError(400, err)
	}
	return c.JSON(http.StatusOK, u)
}

func testKinesis(c echo.Context) (err error) {
	var awsConnect *AWSKinesis

	awsConnect = &AWSKinesis{
		stream:          os.Getenv("KINESIS_STREAM_NAME"),
		region:          os.Getenv("KINESIS_REGION"),
		endpoint:        os.Getenv("AWS_ENDPOINT"),
		accessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		secretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		sessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}

	streamName := &awsConnect.stream

	s := session.New(&aws.Config{
		Region:      aws.String(awsConnect.region),
		Endpoint:    aws.String(awsConnect.endpoint),
		Credentials: credentials.NewStaticCredentials(awsConnect.accessKeyID, awsConnect.secretAccessKey, awsConnect.sessionToken),
	})

	kc := kinesis.New(s)

	record, err := kc.PutRecord(&kinesis.PutRecordInput{
		Data:         []byte(string("HELLO WORLD!")),
		StreamName:   streamName,
		PartitionKey: aws.String("key1"),
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", record)

	if err := kc.WaitUntilStreamExists(&kinesis.DescribeStreamInput{StreamName: streamName}); err != nil {
		panic(err)
	}

	streams, err := kc.DescribeStream(&kinesis.DescribeStreamInput{StreamName: streamName})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", streams)

	putOutput, err := kc.PutRecord(&kinesis.PutRecordInput{
		Data:         []byte("hoge"),
		StreamName:   streamName,
		PartitionKey: aws.String("key1"),
	})

	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", putOutput)

	return c.NoContent(http.StatusOK)
}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func main() {
	godotenv.Load()
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Validator = &CustomValidator{validator: validator.New()}

	e.GET("/", hello)
	e.POST("/users", getUser)
	e.POST("/kinesis", testKinesis)

	e.Logger.Fatal(e.Start(":1323"))
}

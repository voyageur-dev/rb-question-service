package main

import (
	"context"
	"encoding/json"
	"fmt"
	"function/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	getQuestionsPath = "GET /rb/questions"
)

var (
	questionsTableName string

	dbClient *dynamodb.Client
)

func init() {
	questionsTableName = os.Getenv("QUESTIONS_TABLE_NAME")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	path := request.RouteKey

	return func() (events.APIGatewayV2HTTPResponse, error) {
		switch path {
		case getQuestionsPath:
			return getQuestions(request)
		default:
			return events.APIGatewayV2HTTPResponse{
				Body:       "Path Not Found",
				StatusCode: http.StatusNotFound,
			}, nil
		}
	}()
}

func getQuestions(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	queryParams := request.QueryStringParameters
	examId, _ := queryParams["examId"]
	lastEvaluatedKey, hasLastEvaluatedKey := queryParams["lastEvaluatedKey"]
	pageSize, _ := queryParams["pageSize"]
	pageSizeNum, _ := strconv.Atoi(pageSize)

	builder := expression.Key("exam_id").Equal(expression.Value(examId))
	if hasLastEvaluatedKey {
		builder = builder.And(expression.Key("question_id").BeginsWith(lastEvaluatedKey))
	}
	expr, _ := expression.NewBuilder().WithKeyCondition(builder).Build()

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(questionsTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     aws.Int32(int32(pageSizeNum)),
	}

	result, err := dbClient.Query(context.TODO(), input)
	if err != nil {
		log.Println(fmt.Sprintf("Error getting questions: %v", err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error getting questions",
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	questions := make([]models.Question, result.Count)
	for i, item := range result.Items {
		s3ImageUrls := make([]string, 0)
		if itemS3ImageUrls, ok := item["s3_image_urls"]; ok {
			s3ImageUrls = make([]string, len(itemS3ImageUrls.(*types.AttributeValueMemberL).Value))
			for j, url := range item["s3_image_urls"].(*types.AttributeValueMemberL).Value {
				s3ImageUrls[j] = url.(*types.AttributeValueMemberS).Value
			}
		}

		options := make([]models.Option, len(item["options"].(*types.AttributeValueMemberL).Value))
		for j, option := range item["options"].(*types.AttributeValueMemberL).Value {
			optionS3ImageUrls := make([]string, 0)
			if itemOptionS3ImageUrls, ok := option.(*types.AttributeValueMemberM).Value["s3_image_urls"]; ok {
				optionS3ImageUrls = make([]string, len(itemOptionS3ImageUrls.(*types.AttributeValueMemberL).Value))
				for k, url := range itemOptionS3ImageUrls.(*types.AttributeValueMemberL).Value {
					optionS3ImageUrls[k] = url.(*types.AttributeValueMemberS).Value
				}
			}

			options[j] = models.Option{
				IsCorrect:   option.(*types.AttributeValueMemberM).Value["is_correct"].(*types.AttributeValueMemberBOOL).Value,
				Text:        option.(*types.AttributeValueMemberM).Value["text"].(*types.AttributeValueMemberS).Value,
				S3ImageURLs: optionS3ImageUrls,
			}
		}

		questionId, _ := strconv.Atoi(item["question_id"].(*types.AttributeValueMemberN).Value)
		questions[i] = models.Question{
			ExamID:      item["exam_id"].(*types.AttributeValueMemberS).Value,
			QuestionID:  questionId,
			Options:     options,
			Question:    item["question"].(*types.AttributeValueMemberS).Value,
			S3ImageURLs: s3ImageUrls,
		}
	}

	response, _ := json.Marshal(models.GetQuestionsResponse{
		Questions: questions,
	})

	return events.APIGatewayV2HTTPResponse{
		Body:       string(response),
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handler)
}

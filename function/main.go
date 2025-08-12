package main

import (
	"context"
	"encoding/json"
	"fmt"
	"function/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	providerId, _ := queryParams["providerId"]
	examId, _ := queryParams["examId"]
	lastEvaluatedKey, hasLastEvaluatedKey := queryParams["lastEvaluatedKey"]
	pageSize, _ := queryParams["pageSize"]
	pageSizeNum, _ := strconv.Atoi(pageSize)

	providerExamKey := fmt.Sprintf("%s#%s", providerId, examId)
	builder := expression.Key("providerExamKey").Equal(expression.Value(providerExamKey))
	expr, _ := expression.NewBuilder().WithKeyCondition(builder).Build()

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(questionsTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     aws.Int32(int32(pageSizeNum)),
	}

	if hasLastEvaluatedKey {
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"providerExamKey": &types.AttributeValueMemberS{Value: providerExamKey},
			"questionId":      &types.AttributeValueMemberS{Value: lastEvaluatedKey},
		}
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
		log.Println(item)
		description := make([]models.Item, 0)
		if descriptionItems, ok := item["description"]; ok {
			description = make([]models.Item, len(descriptionItems.(*types.AttributeValueMemberL).Value))
			for j, descriptionItem := range descriptionItems.(*types.AttributeValueMemberL).Value {
				log.Println(descriptionItem)
				description[j] = models.Item{
					Content: descriptionItem.(*types.AttributeValueMemberM).Value["content"].(*types.AttributeValueMemberS).Value,
					Type:    descriptionItem.(*types.AttributeValueMemberM).Value["type"].(*types.AttributeValueMemberS).Value,
				}
			}
		}

		answer := make([]models.Item, 0)
		if answerItems, ok := item["answer"]; ok {
			answer = make([]models.Item, len(answerItems.(*types.AttributeValueMemberL).Value))
			for j, answerItem := range answerItems.(*types.AttributeValueMemberL).Value {
				answer[j] = models.Item{
					Content: answerItem.(*types.AttributeValueMemberM).Value["content"].(*types.AttributeValueMemberS).Value,
					Type:    answerItem.(*types.AttributeValueMemberM).Value["type"].(*types.AttributeValueMemberS).Value,
				}
			}
		}

		options := make([]models.Option, len(item["options"].(*types.AttributeValueMemberL).Value))
		for j, option := range item["options"].(*types.AttributeValueMemberL).Value {
			optionDescription := make([]models.Item, 0)
			if optionDescriptionItems, ok := option.(*types.AttributeValueMemberM).Value["description"]; ok {
				optionDescription = make([]models.Item, len(optionDescriptionItems.(*types.AttributeValueMemberL).Value))
				for k, descriptionItem := range optionDescriptionItems.(*types.AttributeValueMemberL).Value {
					optionDescription[k] = models.Item{
						Content: descriptionItem.(*types.AttributeValueMemberM).Value["content"].(*types.AttributeValueMemberS).Value,
						Type:    descriptionItem.(*types.AttributeValueMemberM).Value["type"].(*types.AttributeValueMemberS).Value,
					}
				}
			}

			options[j] = models.Option{
				IsCorrect:   option.(*types.AttributeValueMemberM).Value["isCorrect"].(*types.AttributeValueMemberBOOL).Value,
				Id:          option.(*types.AttributeValueMemberM).Value["id"].(*types.AttributeValueMemberS).Value,
				Description: optionDescription,
			}
		}

		ids := strings.Split(item["providerExamKey"].(*types.AttributeValueMemberS).Value, "#")
		questionId := item["questionId"].(*types.AttributeValueMemberS).Value
		questions[i] = models.Question{
			ProviderId:  ids[0],
			ExamID:      ids[1],
			QuestionID:  questionId,
			Options:     options,
			Description: description,
			Answer:      answer,
		}
	}

	response, _ := json.Marshal(questions)

	return events.APIGatewayV2HTTPResponse{
		Body:       string(response),
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handler)
}

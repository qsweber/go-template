package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const testTable = "test-records"

// fakeDynamoDB records the inputs it receives and returns the configured outputs.
type fakeDynamoDB struct {
	putInput    *dynamodb.PutItemInput
	putErr      error
	getInput    *dynamodb.GetItemInput
	getOutput   *dynamodb.GetItemOutput
	getErr      error
	updateInput *dynamodb.UpdateItemInput
	updateOut   *dynamodb.UpdateItemOutput
	updateErr   error
}

func (f *fakeDynamoDB) PutItem(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.putInput = params
	if f.putErr != nil {
		return nil, f.putErr
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDynamoDB) GetItem(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.getInput = params
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getOutput, nil
}

func (f *fakeDynamoDB) UpdateItem(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updateInput = params
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateOut, nil
}

func stringValue(t *testing.T, item map[string]types.AttributeValue, name string) string {
	t.Helper()
	value, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("attribute %q is %T, want a string", name, item[name])
	}
	return value.Value
}

func assertUserKey(t *testing.T, key map[string]types.AttributeValue, id string) {
	t.Helper()
	if got := stringValue(t, key, PartitionKeyAttribute); got != "USER#"+id {
		t.Errorf("pk = %q, want %q", got, "USER#"+id)
	}
	if got := stringValue(t, key, SortKeyAttribute); got != UserSortKey {
		t.Errorf("sk = %q, want %q", got, UserSortKey)
	}
}

func userItem(id string, displayName string) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		PartitionKeyAttribute: &types.AttributeValueMemberS{Value: UserPartitionKey(id)},
		SortKeyAttribute:      &types.AttributeValueMemberS{Value: UserSortKey},
		"id":                  &types.AttributeValueMemberS{Value: id},
		"created_at":          &types.AttributeValueMemberN{Value: "1700000000000"},
	}
	if displayName != "" {
		item["display_name"] = &types.AttributeValueMemberS{Value: displayName}
	}
	return item
}

func TestUserPartitionKey(t *testing.T) {
	if got := UserPartitionKey("abc"); got != "USER#abc" {
		t.Errorf("UserPartitionKey = %q, want %q", got, "USER#abc")
	}
}

func TestCreateUser(t *testing.T) {
	client := &fakeDynamoDB{}
	repo := NewUsersRepository(testTable, client)

	if err := repo.CreateUser(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := client.putInput
	if aws.ToString(input.TableName) != testTable {
		t.Errorf("TableName = %q, want %q", aws.ToString(input.TableName), testTable)
	}
	assertUserKey(t, input.Item, "user-1")
	if got := stringValue(t, input.Item, "id"); got != "user-1" {
		t.Errorf("id = %q, want %q", got, "user-1")
	}
	if _, ok := input.Item["created_at"].(*types.AttributeValueMemberN); !ok {
		t.Errorf("created_at is %T, want a number", input.Item["created_at"])
	}
	// display_name is user-provided and must not be written (even empty) on creation.
	if _, ok := input.Item["display_name"]; ok {
		t.Error("display_name should not be set on creation")
	}
	if got := aws.ToString(input.ConditionExpression); got != "attribute_not_exists(pk)" {
		t.Errorf("ConditionExpression = %q, want attribute_not_exists(pk)", got)
	}
}

func TestCreateUser_AlreadyExistsIsNotAnError(t *testing.T) {
	client := &fakeDynamoDB{putErr: &types.ConditionalCheckFailedException{}}
	repo := NewUsersRepository(testTable, client)

	if err := repo.CreateUser(context.Background(), "user-1"); err != nil {
		t.Errorf("expected nil error for an existing user, got %v", err)
	}
}

func TestCreateUser_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("boom")
	repo := NewUsersRepository(testTable, &fakeDynamoDB{putErr: wantErr})

	if err := repo.CreateUser(context.Background(), "user-1"); !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestGetUser(t *testing.T) {
	client := &fakeDynamoDB{getOutput: &dynamodb.GetItemOutput{Item: userItem("user-1", "Quinn")}}
	repo := NewUsersRepository(testTable, client)

	user, err := repo.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if aws.ToString(client.getInput.TableName) != testTable {
		t.Errorf("TableName = %q, want %q", aws.ToString(client.getInput.TableName), testTable)
	}
	assertUserKey(t, client.getInput.Key, "user-1")

	want := User{PK: "USER#user-1", SK: "USER", ID: "user-1", DisplayName: "Quinn", CreatedAt: 1700000000000}
	if user == nil || *user != want {
		t.Errorf("user = %+v, want %+v", user, want)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	repo := NewUsersRepository(testTable, &fakeDynamoDB{getOutput: &dynamodb.GetItemOutput{}})

	user, err := repo.GetUser(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("user = %+v, want nil", user)
	}
}

func TestGetUser_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("boom")
	repo := NewUsersRepository(testTable, &fakeDynamoDB{getErr: wantErr})

	if _, err := repo.GetUser(context.Background(), "user-1"); !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestUpdateDisplayName_Set(t *testing.T) {
	client := &fakeDynamoDB{updateOut: &dynamodb.UpdateItemOutput{Attributes: userItem("user-1", "New Name")}}
	repo := NewUsersRepository(testTable, client)

	user, err := repo.UpdateDisplayName(context.Background(), "user-1", "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := client.updateInput
	assertUserKey(t, input.Key, "user-1")
	if got := aws.ToString(input.UpdateExpression); got != "SET #display_name = :display_name" {
		t.Errorf("UpdateExpression = %q", got)
	}
	if got := stringValue(t, input.ExpressionAttributeValues, ":display_name"); got != "New Name" {
		t.Errorf(":display_name = %q, want %q", got, "New Name")
	}
	// Updating must never create a record that the sign-up trigger didn't.
	if got := aws.ToString(input.ConditionExpression); got != "attribute_exists(pk)" {
		t.Errorf("ConditionExpression = %q, want attribute_exists(pk)", got)
	}
	if input.ReturnValues != types.ReturnValueAllNew {
		t.Errorf("ReturnValues = %q, want ALL_NEW", input.ReturnValues)
	}
	if user == nil || user.DisplayName != "New Name" {
		t.Errorf("user = %+v, want display name %q", user, "New Name")
	}
}

func TestUpdateDisplayName_EmptyRemovesAttribute(t *testing.T) {
	client := &fakeDynamoDB{updateOut: &dynamodb.UpdateItemOutput{Attributes: userItem("user-1", "")}}
	repo := NewUsersRepository(testTable, client)

	user, err := repo.UpdateDisplayName(context.Background(), "user-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := client.updateInput
	if got := aws.ToString(input.UpdateExpression); got != "REMOVE #display_name" {
		t.Errorf("UpdateExpression = %q, want REMOVE #display_name", got)
	}
	// DynamoDB rejects unused expression attribute values.
	if input.ExpressionAttributeValues != nil {
		t.Errorf("ExpressionAttributeValues = %v, want nil", input.ExpressionAttributeValues)
	}
	if user == nil || user.DisplayName != "" {
		t.Errorf("user = %+v, want empty display name", user)
	}
}

func TestUpdateDisplayName_NotFound(t *testing.T) {
	repo := NewUsersRepository(testTable, &fakeDynamoDB{updateErr: &types.ConditionalCheckFailedException{}})

	user, err := repo.UpdateDisplayName(context.Background(), "missing", "Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("user = %+v, want nil", user)
	}
}

func TestUpdateDisplayName_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("boom")
	repo := NewUsersRepository(testTable, &fakeDynamoDB{updateErr: wantErr})

	if _, err := repo.UpdateDisplayName(context.Background(), "user-1", "Name"); !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

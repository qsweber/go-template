package repositories

// The app uses a single-table DynamoDB design: every entity lives in the one
// "<stack>-records" table, distinguished by its partition key (pk) and sort key (sk).
const (
	PartitionKeyAttribute = "pk"
	SortKeyAttribute      = "sk"
)

type Repositories struct {
	Users UsersRepository
}

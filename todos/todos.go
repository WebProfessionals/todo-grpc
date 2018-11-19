package todos

import (
	"../protos"
	"github.com/oklog/ulid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/rand"
	"strconv"
	"time"
	"upper.io/db.v3"
)

// Gib mir eine DB Session und du bekommst die impl
func Register(database db.Database) todo.TodoServiceServer {
	dbCollectionTodo = database.Collection("todo")
	var s server
	return &s
}

var dbCollectionTodo db.Collection

// server is used to implement todo.TodoServiceServer.
type server struct {
}

type Hateoas struct {
	links []*todo.Link
}

func (h *Hateoas) AddLink(rel, contenttype, href string, method todo.Link_Method) {
	self := todo.Link{Rel: rel, Href: href, Type: contenttype, Method: method}
	h.links = append(h.links, &self)
}

func (s *server) CreateTodo(ctx context.Context, req *todo.CreateTodoRequest) (*todo.CreateTodoResponse, error) {
	id := GenerateULID().String()
	var item todo.Todo
	item.Id = id
	item.Title = req.Item.Title
	item.Description = req.Item.Description
	if req.Item.Completed != 0 {
		item.Completed = req.Item.Completed
	} else {
		item.Completed = 1
	}
	_, err := dbCollectionTodo.Insert(&item)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not create entity: %s", err)
	}
	e := makeTodoEntity(item)
	return &todo.CreateTodoResponse{Links: e.Links}, nil
}

func (s *server) DeleteTodo(ctx context.Context, req *todo.DeleteTodoRequest) (*todo.DeleteTodoResponse, error) {
	var item todo.Todo
	res := dbCollectionTodo.Find(db.Cond{"id": req.Id})
	if err := res.One(&item); err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not retrieve entity from the database: %s", err)
	}

	if err := res.Delete(); err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not delete entity from the database: %s", err)
	}
	return &todo.DeleteTodoResponse{}, nil
}

func (s *server) UpdateTodo(ctx context.Context, req *todo.UpdateTodoRequest) (*todo.UpdateTodoResponse, error) {

	var item todo.Todo
	res := dbCollectionTodo.Find(db.Cond{"id": req.Id})
	if err := res.One(&item); err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not retrieve entity from the database: %s", err)
	}
	// fields update
	item.Title = req.Item.Title
	item.Description = req.Item.Description
	item.Completed = req.Item.Completed

	err := res.Update(&item)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not update entity: %s", err)
	}
	entity := makeTodoEntity(item)
	return &todo.UpdateTodoResponse{Data: entity.Data, Links: entity.Links}, nil
}

func (s *server) GetTodo(ctx context.Context, req *todo.GetTodoRequest) (*todo.TodoEntity, error) {
	var item todo.Todo
	res := dbCollectionTodo.Find(db.Cond{"id": req.Id})
	if err := res.One(&item); err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not retrieve entity from the database: %s", err)
	}

	entity := makeTodoEntity(item)
	return &entity, nil
}

func (s *server) ListTodo(ctx context.Context, req *todo.ListTodoRequest) (*todo.TodoCollection, error) {
	var items []todo.Todo
	var limit uint = 23
	if req.Limit > 0 {
		limit = uint(req.Limit)
	}
	var page uint = 1
	if req.Page > 0 {
		page = uint(req.Page)
	}
	// todo pagination als methode generalisieren
	nextPage := strconv.FormatUint(uint64(page+1), 10)
	currentPage := strconv.FormatUint(uint64(page), 10)
	prevPage := strconv.FormatUint(uint64(page-1), 10)

	res := dbCollectionTodo.Find().Paginate(uint(limit))

	totalNumberOfPages, _ := res.TotalPages()
	lastPage := strconv.FormatUint(uint64(totalNumberOfPages), 10)

	if err := res.Page(uint(page)).OrderBy("-id").All(&items); err != nil {
		return nil, status.Errorf(codes.NotFound, "Could not retrieve item from the database: %s", err)
	}
	var collection []*todo.TodoEntity
	for _, item := range items {
		entity := makeTodoEntity(item)
		collection = append(collection, &entity)
	}
	var h Hateoas
	h.AddLink("self", "application/json", "http://localhost:8080/todos?page="+currentPage, todo.Link_GET)
	if page > 1 {
		h.AddLink("prev", "application/json", "http://localhost:8080/todos?page="+prevPage, todo.Link_GET)
	}
	if page < totalNumberOfPages {
		h.AddLink("next", "application/json", "http://localhost:8080/todos?page="+nextPage, todo.Link_GET)
	}
	h.AddLink("first", "application/json", "http://localhost:8080/todos?page=1", todo.Link_GET)
	h.AddLink("last", "application/json", "http://localhost:8080/todos?page="+lastPage, todo.Link_GET)
	h.AddLink("create", "application/json", "http://localhost:8080/todos", todo.Link_POST)
	return &todo.TodoCollection{Data: collection, Links: h.links}, nil
}

// erzeuge aus einem db todo item eine todoEntity
func makeTodoEntity(item todo.Todo) todo.TodoEntity {
	var h Hateoas
	h.AddLink("self", "application/json", "http://localhost:8080/todos/"+item.Id, todo.Link_GET)
	h.AddLink("delete", "application/json", "http://localhost:8080/todos/"+item.Id, todo.Link_DELETE)
	h.AddLink("update", "application/json", "http://localhost:8080/todos/"+item.Id, todo.Link_PATCH)
	h.AddLink("up", "application/json", "http://localhost:8080/todos", todo.Link_GET)
	entity := todo.TodoEntity{Data: &item, Links: h.links}
	return entity
}

// Erzeuge eine ULID
func GenerateULID() ulid.ULID {
	t := time.Now().UTC()
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	newID, _ := ulid.New(ulid.Timestamp(t), entropy)
	return newID
}

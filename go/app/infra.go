package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	// STEP 5-1: uncomment this line
	_ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")
var db *sql.DB

type Item struct {
	ID       int    `db:"id" json:"-"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	Image    string `db:"image" json:"image"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetItems(ctx context.Context) ([]Item, error)
	GetItemID(ctx context.Context, itemID int) (Item, error)
	SearchItems(ctx context.Context, keyword string) ([]Item, error)
}

func (i *itemRepository) GetItemID(ctx context.Context, itemID int) (Item, error) {
	var item Item
	err := i.db.QueryRowContext(ctx, "SELECT id, name, category, image_name FROM items WHERE id = ?", itemID).
		Scan(&item.ID, &item.Name, &item.Category, &item.Image)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, fmt.Errorf("item not found: %w", err)
		}
		return Item{}, fmt.Errorf("failed to retrieve item: %w", err)
	}
	return item, nil
}

func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	_, err := i.db.ExecContext(ctx, `
		INSERT INTO items (name, category, image_name) VALUES (?, ?, ?)
	`, item.Name, item.Category, item.Image)
	if err != nil {
		return fmt.Errorf("failed to insert item: %w", err)
	}
	return nil
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
	db       *sql.DB
}

// GetItems implements ItemRepository.
func (i *itemRepository) GetItems(ctx context.Context) ([]Item, error) {
	rows, err := i.db.QueryContext(ctx, "SELECT id, name, category, image_name FROM items")
	if err != nil {
		return nil, fmt.Errorf("failed to get items from database: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (i *itemRepository) SearchItems(ctx context.Context, keyword string) ([]Item, error) {
	query := `
		SELECT id, name, category, image_name 
		FROM items 
		WHERE name LIKE ? OR category LIKE ?
	`
	rows, err := i.db.QueryContext(ctx, query, "%"+keyword+"%", "%"+keyword+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search items from database: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// NewItemRepository creates a new itemRepository.

func NewItemRepository() ItemRepository {
	dbPath, found := os.LookupEnv("DB_PATH")
	if !found {
		dbPath = "db/mercari.sqlite3"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to connect to SQLite: %v", err)
	}

	return &itemRepository{db: db}
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	imageDir := "images"

	if _, err := os.Stat(imageDir); os.IsNotExist(err) {
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			return err
		}
	}

	hash := sha256.Sum256(image)
	hashString := hex.EncodeToString(hash[:])

	filePath := filepath.Join(imageDir, hashString+".jpg")

	if _, err := os.Stat(filePath); err == nil {
		// すでに存在する場合は成功扱い（何もエラーを返さない）
		return nil
	}

	err := os.WriteFile(filePath, image, 0666)
	if err != nil {
		return err
	}
	return nil
}

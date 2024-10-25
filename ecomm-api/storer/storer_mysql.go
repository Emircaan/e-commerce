package storer

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type MySQLStorer struct {
	db *sqlx.DB
}

func NewMySqlStorer(db *sqlx.DB) *MySQLStorer {
	return &MySQLStorer{db: db}
}

func (ms *MySQLStorer) CreateProduct(ctx context.Context, p *Product) (*Product, error) {

	res, err := ms.db.NamedExecContext(ctx, "INSERT INTO products (name, image, category, description, rating, num_reviews, price, count_in_stock) VALUES (:name, :image, :category, :description, :rating, :num_reviews, :price, :count_in_stock)", p)
	if err != nil {
		return nil, fmt.Errorf("could not create product: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert id: %w", err)
	}
	p.ID = id
	return p, nil

}

func (ms *MySQLStorer) GetProduct(ctx context.Context, id int64) (*Product, error) {
	var p Product

	err := ms.db.GetContext(ctx, &p, "SELECT * FROM products WHERE id=?", id)
	if err != nil {
		return nil, fmt.Errorf("could not get product: %w", err)
	}
	return &p, nil
}

func (ms *MySQLStorer) ListProducts(ctx context.Context) ([]*Product, error) {
	var p []*Product
	err := ms.db.SelectContext(ctx, &p, "SELECT * FROM products")
	if err != nil {
		return nil, fmt.Errorf("could not list products: %w", err)
	}
	return p, nil
}

func (ms *MySQLStorer) UpdateProduct(ctx context.Context, p *Product) (*Product, error) {
	_, err := ms.db.NamedExecContext(ctx, "UPDATE products SET name=:name, image=:image, category=:category, description=:description, rating=:rating, num_reviews=:num_reviews, price=:price, count_in_stock=:count_in_stock WHERE id=:id", p)
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	return p, nil
}

func (ms *MySQLStorer) DeleteProduct(ctx context.Context, id int64) error {
	_, err := ms.db.ExecContext(ctx, "DELETE FROM products WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("could not delete product: %w", err)
	}
	return nil
}

func (ms *MySQLStorer) CreateOrder(ctx context.Context, o *Order) (*Order, error) {
	err := ms.execTx(ctx, func(tx *sqlx.Tx) error {
		order, err := createOrder(ctx, tx, o)
		if err != nil {
			return fmt.Errorf("could not create order: %w", err)
		}
		for _, oi := range o.Items {
			oi.OrderID = order.ID
			err = createOrderItem(ctx, tx, oi)
			if err != nil {
				return fmt.Errorf("could not create order item: %w", err)
			}
		}
		return nil

	})
	if err != nil {
		return nil, fmt.Errorf("could not create order: %w", err)
	}
	return o, nil

}

func (ms *MySQLStorer) GetOrder(ctx context.Context, id int64) (*Order, error) {
	var o Order
	err := ms.db.GetContext(ctx, &o, "SELECT * FROM orders WHERE id=?", id)
	if err != nil {
		return nil, fmt.Errorf("could not get order: %w", err)
	}
	var items []OrderItem
	err = ms.db.SelectContext(ctx, &items, "SELECT * FROM order_items WHERE order_id=?", id)
	if err != nil {
		return nil, fmt.Errorf("could not get order items: %w", err)
	}
	o.Items = items
	return &o, nil
}

func (ms *MySQLStorer) ListOrders(ctx context.Context) ([]*Order, error) {
	var orders []*Order
	err := ms.db.SelectContext(ctx, &orders, "SELECT * FROM orders")
	if err != nil {
		return nil, fmt.Errorf("could not list orders: %w", err)
	}
	for i := range orders {
		var items []OrderItem
		err = ms.db.SelectContext(ctx, &items, "SELECT * FROM order_items WHERE order_id=?", orders[i].ID)
		if err != nil {
			return nil, fmt.Errorf("could not get order items: %w", err)
		}
		orders[i].Items = items
	}

	return orders, nil
}

func (ms *MySQLStorer) DeleteOrder(ctx context.Context, id int64) error {
	_, err := ms.db.ExecContext(ctx, "DELETE FROM orders_items WHERE order_id", id)
	if err != nil {
		return fmt.Errorf("could not delete order: %w", err)
	}
	_, err = ms.db.ExecContext(ctx, "DELETE FROM orders WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("could not delete order: %w", err)
	}
	return nil

}

func createOrder(ctx context.Context, tx *sqlx.Tx, o *Order) (*Order, error) {

	res, err := tx.NamedExecContext(ctx, "INSERT INTO orders (payment_method, tax_price, shipping_price, total_price) VALUES (:payment_method, :tax_price, :shipping_price, :total_price)", o)
	if err != nil {
		return nil, fmt.Errorf("could not create order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert id: %w", err)
	}
	o.ID = id
	return o, nil

}

func createOrderItem(ctx context.Context, tx *sqlx.Tx, oi OrderItem) error {
	res, err := tx.NamedExecContext(ctx, "INSERT INTO order_items (name, quantity, image, price, product_id, order_id) VALUES (:name, :quantity, :image, :price, :product_id, :order_id)", oi)
	if err != nil {
		return fmt.Errorf("error inserting order item: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert ID: %w", err)
	}
	oi.ID = id

	return nil
}

func (ms *MySQLStorer) execTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := ms.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

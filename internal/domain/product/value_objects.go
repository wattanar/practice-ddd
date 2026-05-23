package product

type ProductID struct {
	value string
}

func NewProductID(value string) ProductID {
	return ProductID{value: value}
}

func (id ProductID) String() string { return id.value }

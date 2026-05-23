package customer

type CustomerID struct {
	value string
}

func NewCustomerID(value string) CustomerID {
	return CustomerID{value: value}
}

func (id CustomerID) String() string { return id.value }

type Email struct {
	value string
}

func NewEmail(value string) Email {
	return Email{value: value}
}

func (e Email) String() string { return e.value }

type CustomerName struct {
	first string
	last  string
}

func NewCustomerName(first, last string) CustomerName {
	return CustomerName{first: first, last: last}
}

func (n CustomerName) First() string     { return n.first }
func (n CustomerName) Last() string      { return n.last }
func (n CustomerName) FullName() string  { return n.first + " " + n.last }

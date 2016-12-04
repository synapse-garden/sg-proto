package store

// Resource can be used for package constants, etc. to name a resource.
type Resource string

// Resourcer is something which has a Resource.
type Resourcer interface {
	Resource() Resource
}

// ResourceBox is a container for the bytes and type of some encoded
// Resource.
type ResourceBox struct {
	Name     Resource `json:"name"`
	Contents string   `json:"contents"`
}

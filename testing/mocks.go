package testing

//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ./mocks/io.go -mock_names Reader=Reader,Writer=Writer io Reader,Writer

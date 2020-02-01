package main

import (
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"github.com/stripe/stripe-go/client"
)

func main() {
	rootFields := graphql.Fields{}

	gqlMap := map[string]graphql.Type{
		"string":  graphql.String,
		"int64":   graphql.Int,
		"bool":    graphql.Boolean,
		"float64": graphql.Float,
	}

	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" {
		panic("Please set the STRIPE_SECRET_KEY environment variable")
	}

	client := client.New(secretKey, nil)
	re := reflect.Indirect(reflect.ValueOf(client))

	for i := 0; i < re.NumField(); i++ {
		fieldType := re.Type().Field(i)
		field := re.Field(i)

		fields := graphql.Fields{}

		getMethod, ok := field.Type().MethodByName("Get")
		if !ok || getMethod.Type.NumIn() != 3 {
			fmt.Println("could not get 'Get' method of: ", fieldType.Name)
			continue
		}

		out := getMethod.Type.Out(0).Elem()

		for j := 0; j < out.NumField(); j++ {
			field := out.Field(j)
			gqlType := gqlMap[field.Type.Name()]
			if gqlType != nil {
				fields[field.Name] = &graphql.Field{
					Type: gqlType,
				}
			}
		}

		if len(fields) == 0 {
			continue
		}

		gqlType := graphql.NewObject(graphql.ObjectConfig{
			Name:   fieldType.Name,
			Fields: fields,
		})

		rootFields[fieldType.Name] = &graphql.Field{
			Type:        gqlType,
			Description: "",
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.String,
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				id, ok := params.Args["id"].(string)
				if !ok {
					return nil, fmt.Errorf("couldn't cast id to string")
				}

				returns := getMethod.Func.Call([]reflect.Value{field, reflect.ValueOf(id), reflect.New(getMethod.Type.In(2).Elem())})

				ret1 := returns[0].Interface()
				var ret2 error
				if !returns[1].IsNil() {
					ret2 = returns[1].Interface().(error)
					ret1 = nil
				}
				return ret1, ret2
			},
		}
	}

	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "RootQuery",
			Fields: rootFields,
		}),
	})

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.Handle("/graphql", h)
	http.ListenAndServe(":8080", nil)
}

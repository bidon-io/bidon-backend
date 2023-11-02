package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) < 2 {
		flags.Usage()
		return
	}

	command := args[0]

	db, err := goose.OpenDBWithDriver("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open DB: %v\n", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("failed to close DB: %v\n", err)
		}
	}()

	arguments := make([]string, 0)
	if len(args) > 2 {
		arguments = append(arguments, args[2:]...)
	}

	fmt.Println("args: ", args)
	fmt.Println("arguments: ", arguments)
	if err := goose.Run(command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose: %v", err)
	}
}

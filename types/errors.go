package types

import (
	"fmt"
	"log"
	"runtime/debug"
)

func CheckError(err error) {

	if err != nil {
		fmt.Println(string(debug.Stack()))
		log.Fatalln("Error has occured:", err.Error())
	}

}

package main

import (
	"fmt"

	"github.com/ppreeper/odoosearchdomain"
)

var patterns = []string{
	// "",
	// " ",
	// "[]",
	// " []",
	// "[] ",
	// " [] ",
	// "[ ]",
	// " [ ]",
	// "[ ] ",
	// " [ ] ",
	// "('')",
	// "('','')",
	// "('a','=')",
	// "('name')",
	// "('name','=')",
	// "[()]",
	// "[(]",
	// "[)]",
	// "[('name','=','My Name')]",
	// "[('name','=',123)]",
	// "[('name','=',1.23)]",
	// "[('name','=',True)]",
	// "[('name','=',False)]",
	// "[('name','=',None)]",
	// "[('name','=',true)]",
	// "[('name','=',false)]",
	// "[('name','=',none)]",
	// "[('name','=')]",
	// "[('name','=','')]",
	// "['!',('name','=','')]",
	// "[('name','=','Peter (Dad) Preeper')]",
	// "[('a','=','b'),('c','=','d')]",
	"['|',('a','=','b'),('c','=','d')]",
	"[('a','=','b (test)'),('c','in',['d'])]",
	// "[('name','like','My Name')]",
	// "[('name','ilike','My Name')]",
	// "[('name','=','My Name'),('ref','=',12345)]",
	// "[('name','=','My Name'),('amount','=',123.45)]",
	// "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),('customer','=',True)]",
	// "[('name', '=', 'John'), '|', ('is_company', '=', True), ('customer', '=', True)]",
	// "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'&', ('is_company', '=', True),('customer','=',True)]",
	// "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]",
	// "[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]",
	// "[('name','lik','My Name')]",
	// "[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]",
	// "[('name', '=', 'ABC'), '|', ('phone','ilike','7620'),'|']",
	// "[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]",
	// "[('name', '=', 'ABC'), '!']",
	// "['|','&',('parent_id','=',False),('id','=',partner_id), '&',('parent_id','=',partner_id),('type','=','invoice')]",
	// "['|','&',('parent_id','=',False),('id','=',partner_id), '&',('parent_id','=',partner_id),('type','=','delivery')]",
	// "[('birthday.month_number', 'in', [10,11,12])]",
	// "[('birthday.month', 'in', ['April','May','June'])]",
	// "[('birthday.month_number', 'in', [10,11,12]), ('birthday.month', 'in', ['April','May','June'])]",
	// "[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [('product_id.qty_available', '<=', 0)])]",
	// "[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [ ('product_id.qty_available', '<=', 0) , ('name','ilike','stud')])]",
}

func main() {
	for _, pattern := range patterns {
		fmt.Println("--------------------------------------------------")
		fmt.Printf("\nParsing domain pattern: '%s'\n", pattern)
		filter, err := odoosearchdomain.Parse(pattern)
		if err != nil {
			fmt.Printf("Error parsing domain '%s': %v\n", pattern, err)
			continue
		}
		fmt.Printf("Parsed domain '%s' to filter: %v\n", pattern, filter)
		fmt.Println()
	}
}

// vi: sw=4 ts=4:
/*
 ---------------------------------------------------------------------------
   Copyright (c) 2013-2016 AT&T Intellectual Property

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at:

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 ---------------------------------------------------------------------------
*/


/*

	Mnemonic:	jsontree
	Abstract:	This is basically a set of wrappers which unmarshal the raw json
				blob into a jif, and then allow the caller to 'extract' values
				with a simple j.Get_xxx( name ) style call.
	Date:		04 April 2016
	Author:		E. Scott Daniels

	Mods:		21 March 2018 - Added pretty print function
*/

package jsontools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

/*
	We return a simple struct which points to the interface that was generated by the unmarshal
	process.  This allows for object function calls rather than passing the object as parm 1.
*/
type Jtree struct {
	jmap		map[string]interface{}
}

// --------------------- public ----------------------------------------------------------------------

/*
	Given the 'raw' json interface created by unmarshal, convert it into our base 
	struct and return a pointer to the struct.
*/
func Json2tree( json_blob []byte ) ( j *Jtree, err error ) {
	var (
		jif	interface{};				// were go's json will unpack the blob into interface form
	)

	err = json.Unmarshal( json_blob, &jif )			// unpack the json into jif
	if err != nil {
		return nil, fmt.Errorf( "unable to unpack json into jif: %s\n", err )
	}

	if m, ok := jif.( map[string]interface{} ); ok {
		j = &Jtree{ jmap:	m } 
	} else {
			return nil, fmt.Errorf( "pointer to jif map wasn't to a map[string]interface{}" )
	}

	return j, nil
}

/*
	Take the jtree and put it back into a json string (frock it).
*/
func ( j *Jtree ) Frock( ) ( jstr string ) {
	if j == nil {
		return ""
	}

	return Frock_jmap( j.jmap )
}

/*
	Simple field confirmation function. Returns true if the named field
	exists in the json mess.
*/
func ( j *Jtree ) Has_field( name string ) ( bool ) {
	if j == nil {
		return false
	}

	thing := j.jmap[name]
	return thing != nil
}

/*
	Look up name and return a pointer to it if it is a string.
*/
func (j *Jtree ) Get_string( name string ) ( *string ) {
	var(
		st string
		ok bool
	)

	thing := j.jmap[name]
	if thing != nil {
		if st, ok = thing.( string ); !ok {
			return nil
		}	
	} else {
		return nil
	}

	return &st
}

/*
	Look up the name and if it's a float return the value. bool in return 
	is set to true if found.
*/
func (j *Jtree ) Get_float( name string ) ( float64, bool ) {
	var (
		value	float64 = 0.0
		ok 		bool = false
	)

	thing := j.jmap[name]
	if thing != nil {
		ok = true
		switch v := thing.( type ) {
			case float64:
				value = v

			case int:
				value = float64( v )

			case int64:
				value = float64( v )

			default:
				ok = false
		}
	}

	return value, ok
}

/*
	Look up name and return integer value. We assume unmarshall saves all
	values as float.
*/
func (j *Jtree ) Get_int( name string ) ( int64, bool ) {
	var (
		value	int64 = 0
		ok 		bool = false
	)

	thing := j.jmap[name]
	if thing != nil {
		switch v := thing.( type ) {
			case float64:
				ok = true
				value = int64( v )

			case int:
				ok = true
				value = int64( v )

			case int64:
				ok = true
				value = v

			default:
				ok = false
		}
	}

	return value, ok
}

/*
	Generic getvalue -- returns the value as an interface; caller 
	must figure out type.
*/
func ( j *Jtree ) Get_field_if( ifname interface{} ) ( interface{} ) {
	if j == nil {
		return nil
	}

	switch fname := ifname.(type) {
		case string:
			return j.jmap[fname]

		case *string:
			return j.jmap[*fname]

		default:
			return nil;
	}	
}

/*
	Return the value associated with a boolean; ok is false if the value
	isn't booliean or doesn't exist and the value returned is undefined.
*/
func ( j *Jtree ) Get_bool( name string ) ( bv bool, ok bool ) {
	thing := j.jmap[name]
	if thing == nil {
		return false, false
	}

	bv, ok = thing.( bool )
	return bv, ok					// bv is undefined if !ok
}

func( j *Jtree ) Get_subtree( name string ) ( *Jtree, bool ) {
	var (
		st *Jtree = nil
		m map[string]interface{}
	)

	ok := false

	thing := j.jmap[name]
	if thing != nil {
		if m, ok = thing.( map[string]interface{} ); ok {
			st = &Jtree{ jmap:	m } 
		}
	}

	return st, ok
}

/*
	Return the number of elements in the array name or -1 if not an array.
*/
func ( j *Jtree ) Get_ele_count( name string ) ( int ) {
	
	thing := j.jmap[name]
	if thing == nil {
		return -1
	}
	
	a, ok := thing.( []interface{} )
	if ! ok {
		return -1
	}

	return len( a )
}

/*
	Return the *string element from the array. Nil is returned if:
		name isn't defined in the tree OR
		name isn't an array OR
		idx is out of range for name OR
		a[idx] isn't a string
		
*/
func ( j *Jtree ) Get_ele_string( name string, idx int ) ( *string ) {
	ele := j.Get_ele_if( name, idx )
	if ele == nil {
		return nil
	}

	st, ok := (*ele).( string ); if !ok {
		return nil
	}

	return &st
}

/*
	Return the value associated with a boolean; ok is false if the value
	isn't booliean or doesn't exist or there is a range error.
*/
func ( j *Jtree ) Get_ele_bool( name string, idx int ) ( bv bool, ok bool ) {
	ele := j.Get_ele_if( name, idx )
	if ele == nil {
		return false, false
	}

	bv, ok = (*ele).( bool )
	return bv, ok					// bv is undefined if !ok
}

/*
	Return the int element from the array. ok is false if:
		name isn't defined in the tree OR
		name isn't an array OR
		idx is out of range for name OR
		a[idx] cannot be converted into an integer
		
*/
func ( j *Jtree ) Get_ele_int( name string, idx int ) ( int64, bool ) {
	var value int64 = 0

	ele := j.Get_ele_if( name, idx )
	if ele == nil {
		return 0, false
	}

	
	ok := true
	switch v := (*ele).( type ) {
		case int:
			value = int64( v )
			
		case int64:
			value = v

		case float64:
			value = int64( v )

		default:
			ok = false
	}

	return  value, ok
}

/*
	Return the float element from the array. ok is false if:
		name isn't defined in the tree OR
		name isn't an array OR
		idx is out of range for name OR
		a[idx] cannot be converted into a float
		
*/
func ( j *Jtree ) Get_ele_float( name string, idx int ) ( float64, bool ) {
	var value float64 = 0.0

	ele := j.Get_ele_if( name, idx )
	if ele == nil {
		return 0.0, false
	}

	
	ok := true
	switch v := (*ele).( type ) {
		case int:
			value = float64( v )
			
		case int64:
			value = float64( v )

		case float64:
			value = v

		default:
			ok = false
	}

	return  value, ok
}

/*
	Return the element of an array as a pointer to a subtree.
	Returns nil and !ok if:
		name is not known OR
		name is not an array OR
		index is out of range OR
		element is not an 'object'
	
*/
func ( j *Jtree ) Get_ele_subtree( name string, idx int ) ( *Jtree, bool ) {

	var (
		st	*Jtree = nil			// subtree
		m	map[string]interface{}
	)

	ok := false

	ele := j.Get_ele_if( name, idx )
	if ele != nil {
		if m, ok = (*ele).( map[string]interface{} ); ok {
			st = &Jtree{ jmap:	m } 
		}
	}

	return st, ok
}

/*
	Get the element in the named array if not out of range.
	Retuns nil if:
		name is not in the tree OR
		name is not an array OR
		idx is out of range
*/
func ( j *Jtree ) Get_ele_if( name string, idx int ) ( *interface{} ) {
	
	thing := j.jmap[name]
	if thing == nil {
		return nil
	}
	
	a, ok := thing.( []interface{} )
	if ! ok {
		return nil
	}

	if idx < 0 || idx > len( a ) {
		return nil
	}

	return &a[idx]
}


/*
	Get_keys returns an array of field names which are available at the current 
	top level of the tree.
*/
func( j *Jtree ) Get_fnames( ) ( fnames []string ) {
	if j == nil || j.jmap == nil {
		return nil 
	}

	fnames = make( []string, len( j.jmap ) )
	i := 0
	for k,_ := range j.jmap {
		fnames[i] = k
		i++
	}

	return fnames
}

/*
	Pretty print the interface recusring to handle array and nested interface things
	which we assume are jtrees.
*/
func print_if( target io.Writer, root string, stuff interface{} ) {
	if stuff == nil {
		return
	}

	switch val := (stuff).(type) {
		case int:
			fmt.Fprintf( target, "%s = %d\n", root, val )

		case float64:
			fmt.Fprintf( target, "%s = %.2f\n", root, val )

		case bool:
			fmt.Fprintf( target, "%s = %v\n", root, val )

		case string:
			fmt.Fprintf( target, "%s = %s\n", root, val )

		case *string:
			fmt.Fprintf( target, "%s = %s\n", root, *val )

		case []interface{}:
			nele := len( val )
			for j := 0; j < nele; j += 1 {
				aroot := fmt.Sprintf( "%s[%02d]", root, j )
				print_if( target, aroot, val[j] )
			}

		case map[string]interface{}:			// subtree
			for k, v := range val {
				sroot := fmt.Sprintf( "%s.%s", root, k )
				print_if( target, sroot, v )
			}

		default:
			fmt.Fprintf( target, "%s = unknown-type\n", root )
	}
}

/*
	Pretty print the json tree. Arrays printed in order, rest is up to the whim of
	the hash function that delivers things from the map.
*/
func ( jt *Jtree ) Pretty_print( target io.Writer  ) {
	if jt == nil {
		return
	}

	fields := jt.Get_fnames()
	for i := 0; i < len( fields ); i += 1 {
		print_if( target, fields[i], jt.Get_field_if( fields[i] ) )
	}
}


// ----  debugging ---------------------------------------------------------
/*
	Generate a list of fields in the current tree.
*/
func (j *Jtree ) List_fields() ( flist string ) {
	flist = ""

	if j == nil {
		return flist
	}

	for key := range j.jmap {
		flist += key + " "
	}
	
	return strings.Trim( flist, " " )
}

/*
	Spill our guts for debugging
*/
func (j *Jtree ) Dump() {
	if j == nil {
		return
	}

	for k,v := range j.jmap {
		fmt.Fprintf( os.Stderr, "dump: key=%s ", k )
		switch val := v.(type) {
			case string:
				fmt.Fprintf( os.Stderr, " <string> %s\n", val )

			case *string:
				fmt.Fprintf( os.Stderr, " <*string> %s\n", *val )

			case float64:
				fmt.Fprintf( os.Stderr, " <value> %.3f\n", val )

			case bool:
				fmt.Fprintf( os.Stderr, " <bool> %v\n", val )

			default:
				t := reflect.TypeOf( val ).Elem()					// not recognised; give type
				fmt.Fprintf( os.Stderr, " type: %s\n", t )
		}
	}
}


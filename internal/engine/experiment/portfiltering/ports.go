package portfiltering

//
// List of ports we want to measure
//

var Ports = []string{
	"22",  // tcp
	"23",  // tcp
	"25",  // tcp
	"80",  // tcp
	"143", // tcp
	"443", // tcp
	"445", // tcp
	"587", // tcp
	"993", // tcp
}

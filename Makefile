resources.c: resources.xml
	glib-compile-resources $< --generate-source --target=$@

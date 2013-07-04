### Example Project For Seven5, "myproj"

#### This directory, myproj/src/myproj

This directory is where the source code for the server side of "myproj" should live. Note that this
could should be a _library_ not an executable.  The result of building this code can be seen in
"../../pkg".  You can build the source code like this:

```
go install myproj
```


If you have trouble building the source code check the "README.md" in "../.." for instructions on
setting your `GOPATH`.

Note that the build command can be run from _any_ directory once your `GOPATH` is set properly.

The executable or entry-point of the project is in the child directory of this one, "runmyproj".


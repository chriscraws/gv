Currently being used mostly for game / interactive application experiments--

[Development workflow in a game](https://www.youtube.com/watch?v=8He97Sl9iy0) (state-preserving code reload incl. crash recovery, reflection-based serialization and scene editor UI)
[Game jam game it was used on](https://github.com/nikki93/raylib-5k) (source included, link to game playable in web (Wasm), same engine as in above video)

# gx, a Go -> C++ compiler

Historically I've used C and C++ for gameplay programming so far, but there are various usability issues with it, especially regarding various dark corners and tangents in the language. With intention I'm able to be aware of and stick to a subset that works well, but generally it's tough to have new programmers pick it up or to establish consensus and best practices across a team (I lead engineering on a team that uses C++ for a game engine, and the language definitely complicates things). This was the reasoning behind making Go itself -- eg. in this talk from Rob Pike https://www.infoq.com/presentations/Go-Google/ (esp. around 11 minutes in)

I went on a bit of a journey diving into Nim, Zig, Rust, Go and a few other things for the same purpose, and each thing was almost there but with its own differences that made it not be exactly what I wanted. Semantically Go was my favorite -- just a great combination of simplification and focus. But the runtime didn't fit the game development scenario so well, especially for WebAssembly which is definitely a target I need. When testing a game on WebAssembly with Go I was hitting GC pauses frequently and the perf was pretty not great. I had to stick with C++ in practice, both for my main project at work and my side projects. I know I can "do things to maybe cause the GC to run less" or such, but then that immediately starts to detract from the goal of having a language where I can focus on just the gameplay code.

Over time I collected some ideas about what I'd like to see in my own language for this purpose, but actually building my own compiler (parser, semantic analysis, codegen etc.) from scratch, and also all of the tools alongside (editor integration, github highlighting its syntax etc.) is just a huge task. So I was also kind of thinking about taking an existing language toolchain and modifying it. Modifying C++ was something I looked into briefly but LLVM's codebase is just not fun. I looked at TypeScript briefly. Some of the other languages are still kind of evolving pretty fast / are early, and that adds design+impl overhead about tracking their evolution. But then I found that Go's parser and typechecker that are already in the standard library had a really nice API. And the 'tools/packages' library goes further and automatically resolves packages with usual Go semantics (including modules) which was perfect. Design-wise it was also perfect I think, by providing a focused core that I can add semantics around as I need. Methods being definable outside structs was just the icing on the cake in terms of alignment with the data-oriented style I was after (func (t Thing) Foo(...) codegens foo(t Thing, ...), which interops really well both with calling to C libraries but also overload-resolution in surrounding C++ code that calls your code).

So I figured I could probably write a compiler to C++ pretty easily since C++ is about at the same level, I don't really have to 'lower' any of the Go constructs, it's just translation. I started on that and it turned out to work pretty soon. All the tools just work since it appears as regular go code: my editor integration autocompletes and shows errors, highlights, GitHub highlighting and cross references work, gofmt works, etc.

That's the background / journey, but to summarize just the reasons:

* **Portability**: C++ runs in a lot of contexts that I care about: WebAssembly, natively on major desktops, and on iOS and Android. This is pretty important for me for game and UI application use cases.
* **Interop**: With this compiler I have direct interop to existing C++ libraries and codebases. There's no added overhead or anything since it actually just directly codegens calls to those libraries in the generated code. All you need to do is put //gx:extern TheCppFunc above a function declaration in Go and you're saying "don't generate this function, and when calling just call TheCppFunc instead." For example I use EnTT for the entity-component data storage in the game demo here, and I'm able to call to the C++ including Go's generic syntax translating to template calls. That's not as easy with Cgo (and also Cgo adds huge overhead). This is often pretty important in game / UI application development.
* **Performance**: This compiler targets a specific subset of Go that I think of as being perfect for data-oriented gameplay code. There's no GC or concurrency. Slices and strings are just value types and their memory is released automatically at the end of a scope or if their containing struct was dropped (this is just C++ behavior). So the perf loss of the GC is just not there. But there's not much of a loss of ergonomics, since game code is usually not a pointer soup, and the ECS is the main way that I store all of the dynamic data. You don't see any 'manual memory management' in any of the game code at all, but there's also no GC. That's just the GC point -- in general there's more control over what code is generated since I have a general sense of what C++ compiles to. eg. the language only supports non-escaping lambdas, and those get usually inlined by clang. This gains all of the optimization of clang+llvm.
* **Control**: The entire compiler is ~1500 lines of Go, which makes it easy for me to make any change as it comes up in practice as I work on the game or other applications. For example, it's useful in my game to track all of the structs that make up components for entities, so that I can deserialize them from JSON. This was a pretty easy feature to add by just adding it to the compiler. Generally speaking all of the 'metaprogramming' things I want to do (of which there are specific things that help in games, mostly related to game content stored as data made by level designers or showing them in editor tools) is pretty straightforward to do with this level of compiler control. In metaprogramming-capable languages like Nim and Zig that I dug into, you still have to orient what you want to do in terms of their metaprogramming model and it doesn't often fit or often isn't even possible. Another example -- this was the entire change needed to add default values for struct fields (granted, it leverages the C++ feature, but that's also kind of the point -- C++ is a great kitchen sink of language features that I can now design in terms of).

The current scope of this is just to use in this game, which is a side project I'm working on with a few friends. The stretch goal later is to make it more of a framework and tool (including the scene editor and entity system) for new programmers to use to dive into gameplay programming in a data oriented style, while incrementally being able to go deeper into engine things and not feel like there's a big dichotomy between some scripting language they use and the underlying engine (this is often how it is with existing engines). You can always mess with the bytes, call to C/C++ things, implement things yourself, etc. But at the same time I want there to be usability things like a scene and property editor, even if you own the data yourself in structs and slices. That's what you see in the [demo video](https://www.youtube.com/watch?v=8He97Sl9iy0).

package main

// Represents something that a work was made with
// used for the /using/_technology path
type Technology struct {
	URLName      string   // (unique) identifier used in the URL
	DisplayName  string   // name displayed to the user
	Aliases      []string // aliases pointing to the canonical URL (built from URLName)
	Author       string   // What company is behind the tech? (to display i.e. 'Adobe Photoshop' instead of 'Photoshop')
	LearnMoreURL string   // The technology's website
	Description  string   // A short description of the technology
}

// ReferredToBy returns whether the given name refers to the tech
func (t *Technology) ReferredToBy(name string) bool {
	return StringsLooselyMatch(name, t.URLName, t.DisplayName) || StringsLooselyMatch(name, t.Aliases...)
}

var KnownTechnologies = [...]Technology{
	{
		URLName:      "aftereffects",
		DisplayName:  "After Effects",
		Author:       "Adobe",
		LearnMoreURL: "https://www.adobe.com/fr/products/aftereffects.html",
		Description:  "A digital visual effects, motion graphics, and compositing application",
	},
	{
		URLName:      "asciidoc",
		DisplayName:  "asciidoc",
		LearnMoreURL: "https://asciidoc.org/",
		Description:  "A text document format for writing notes, documentation, articles, books, ebooks, slideshows, web pages, man pages and blogs. AsciiDoc files can be translated to many formats including HTML, PDF, EPUB, man page.",
	},
	{
		URLName:      "assembly",
		DisplayName:  "Assembly",
		LearnMoreURL: "https://apps.apple.com/us/app/assembly-graphic-design-art/id1024210402",
		Description:  "An intuitive vector graphics editor for iOS",
	},
	{
		URLName:      "coffeescript",
		DisplayName:  "CoffeeScript",
		LearnMoreURL: "https://coffeescript.org/",
		Description:  "A programming language that compiles to JavaScript. It adds syntactic sugar inspired by Ruby, Python and Haskell in an effort to enhance JavaScript's brevity and readability. Specific additional features include list comprehension and destructuring assignment",
	},
	{
		URLName:      "css",
		DisplayName:  "CSS",
		LearnMoreURL: "https://developer.mozilla.org/fr/docs/Web/CSS",
		Description:  "A stylesheet language used to describe the presentation of a document written in HTML or XML",
	},
	{
		URLName:     "c#",
		DisplayName: "C#",
		Aliases:     []string{"cs", "csharp"},
		Author:      "Microsoft",
	},
	{
		URLName:     "c++",
		DisplayName: "C++",
		Aliases:     []string{"cpp", "cplusplus"},
	},
	{
		URLName:     "c",
		DisplayName: "C",
	},
	{
		URLName:      "djangorestframework",
		DisplayName:  "Django REST Framework",
		Author:       "encode",
		LearnMoreURL: "https://www.django-rest-framework.org",
		Description:  "A powerful and flexible toolkit for building Web APIs with Django",
	},
	{
		URLName:      "django",
		DisplayName:  "Django",
		LearnMoreURL: "https://www.djangoproject.com",
		Description:  "A high-level Python Web framework that encourages rapid development and clean, pragmatic design",
	},
	{
		URLName:      "docopt",
		DisplayName:  "Docopt",
		LearnMoreURL: "https://github.com/docopt",
		Description:  "A library that parses command-line arguments based on a help message. “Don't write parser code: a good help message already has all the necessary information in it”",
	},
	{
		URLName:      "figma",
		DisplayName:  "Figma",
		Author:       "Google",
		LearnMoreURL: "https://figma.com",
		Description:  "A collaborative web-based vector editor, UX design and prototyping tool",
	},
	{
		URLName:      "fishshell",
		DisplayName:  "Fish Shell",
		Aliases:      []string{"fish"},
		LearnMoreURL: "https://fishshell.com",
		Description:  "A smart and user-friendly command line shell for Linux, macOS, and the rest of the family",
	},
	{
		URLName:      "flstudio",
		DisplayName:  "FL Studio",
		Aliases:      []string{"fruityloops"},
		Author:       "Image-Line",
		LearnMoreURL: "https://www.image-line.com/flstudio/",
		Description:  "A digital audio workstation (DAW)",
	},
	{
		URLName:      "gimp",
		DisplayName:  "GIMP",
		Author:       "GNU",
		LearnMoreURL: "https://www.gimp.org/",
		Description:  "An open source image editor",
	},
	{
		URLName:      "go",
		DisplayName:  "Go",
		Author:       "Google",
		LearnMoreURL: "https://go.dev",
		Description:  "An straightforward low-level open source programming language supported by Google featuring built-in concurrency and a robust standard library",
	},
	{
		URLName:      "html",
		DisplayName:  "HTML",
		LearnMoreURL: "https://developer.mozilla.org/fr/docs/Web/HTML",
		Description:  "The standard markup language for documents designed to be displayed in a web browser",
	},
	{
		URLName:      "illustrator",
		DisplayName:  "Illustrator",
		Author:       "Adobe",
		LearnMoreURL: "https://www.adobe.com/fr/products/illustrator.html",
		Description:  "A vector graphics editor and design program",
	},
	{
		URLName:      "indesign",
		DisplayName:  "InDesign",
		Author:       "Adobe",
		LearnMoreURL: "https://www.adobe.com/fr/products/indesign.html",
		Description:  "A desktop publishing and typesetting software application used to create works such as posters, flyers, brochures, magazines, newspapers, presentations, books and ebooks",
	},
	{
		URLName:      "javascript",
		DisplayName:  "JavaScript",
		Aliases:      []string{"js"},
		Author:       "Mozilla",
		LearnMoreURL: "https://developer.mozilla.org/fr/docs/Web/JavaScript",
		Description:  "A high-level programming language that enables interactive web pages",
	},
	{
		URLName:      "json",
		DisplayName:  "JSON",
		LearnMoreURL: "https://www.json.org/",
		Description:  "A lightweight data-interchange format",
	},
	{
		URLName:      "latex",
		DisplayName:  "LaTeX",
		LearnMoreURL: "https://www.latex-project.org/",
		Description:  "A high-quality typesetting & document preparation system",
	},
	{
		URLName:      "livescript",
		DisplayName:  "LiveScript",
		LearnMoreURL: "https://livescript.net",
		Description:  "A language which compiles to JavaScript that adds many features to assist in functional style programming. LiveScript is an indirect descendant of CoffeeScript, with which it has much compatibility.",
	},
	{
		URLName:      "lua",
		DisplayName:  "Lua",
		LearnMoreURL: "https://lua.org",
		Description:  "A powerful, efficient, lightweight, embeddable scripting language",
	},
	{
		URLName:      "markdown",
		DisplayName:  "Markdown",
		LearnMoreURL: "https://daringfireball.net/projects/markdown/",
		Description:  "A text-to-HTML conversion tool and language for web writers. The overriding design goal for Markdown’s formatting syntax is to make it as readable as possible: a Markdown-formatted document should be publishable as-is, as plain text, without looking like it’s been marked up with tags or formatting instructions.",
	},
	{
		URLName:      "nestjs",
		DisplayName:  "NestJS",
		LearnMoreURL: "https://nestjs.com/",
		Description:  "A progressive Node.js framework for building efficient, reliable and scalable server-side applications",
	},
	{
		URLName:      "nim",
		DisplayName:  "Nim",
		LearnMoreURL: "https://nim-lang.org/",
		Description:  "A statically typed compiled systems programming language. It combines successful concepts from mature languages like Python, Ada and Modula.",
	},
	{
		URLName:      "nuxt",
		DisplayName:  "Nuxt",
		Aliases:      []string{"nuxtjs"},
		LearnMoreURL: "https://nuxtjs.org",
		Description:  "The Intuitive Vue Framework",
	},
	{
		URLName:      "oclif",
		DisplayName:  "Oclif",
		Author:       "Heroku",
		LearnMoreURL: "https://oclif.io/",
		Description:  "An open Node.js CLI framework",
	},
	{
		URLName:      "photoshop",
		DisplayName:  "Photoshop",
		Author:       "Adobe",
		LearnMoreURL: "https://www.adobe.com/fr/products/photoshop.html",
		Description:  "A raster graphics editor",
	},
	{
		URLName:      "php",
		DisplayName:  "PHP",
		LearnMoreURL: "https://www.php.net/",
		Description:  " A popular general-purpose scripting language that is especially suited to web development",
	},
	{
		URLName:      "plantuml",
		DisplayName:  "PlantUML",
		LearnMoreURL: "https://plantuml.com/",
		Description:  "A language for generating UML diagrams from textual descriptions",
	},
	{
		URLName:      "postgresql",
		DisplayName:  "PostGreSQL",
		LearnMoreURL: "https://www.postgresql.org/",
		Description:  "An advanced open source relational database",
	},
	{
		URLName:      "premierepro",
		DisplayName:  "Premiere Pro",
		Author:       "Adobe",
		LearnMoreURL: "https://www.adobe.com/fr/products/premiere.html",
		Description:  "A timeline-based video editing software application",
	},
	{
		URLName:      "pug",
		DisplayName:  "Pug",
		LearnMoreURL: "https://pugjs.org",
		Description:  "A high-performance template engine heavily influenced by <a href='https://haml.info/'>Haml</a> and implemented with JavaScript for Node.js and browsers.",
	},
	{
		URLName:      "pychemin",
		DisplayName:  "PyChemin",
		Author:       "ewen-lbh",
		LearnMoreURL: "https://ewen.works/pychemin",
		Description:  "A minimal language to describe interactive, terminal-based text adventure games",
	},
	{
		URLName:      "python",
		DisplayName:  "Python",
		Author:       "PSF",
		LearnMoreURL: "https://python.org",
		Description:  "A programming language that lets you work quickly and integrate systems more effectively",
	},
	{
		URLName:      "rubyonrails",
		DisplayName:  "Ruby On Rails",
		LearnMoreURL: "https://rubyonrails.org",
		Description:  "A web-application framework that includes everything needed to create database-backed web applications according to the Model-View-Controller (MVC) pattern",
	},
	{
		URLName:      "ruby",
		DisplayName:  "Ruby",
		LearnMoreURL: "https://www.ruby-lang.org",
		Description:  "A dynamic, open source programming language with a focus on simplicity and productivity. It has an elegant syntax that is natural to read and easy to write.",
	},
	{
		URLName:      "rust",
		DisplayName:  "Rust",
		Author:       "Mozilla",
		LearnMoreURL: "https://www.rust-lang.org",
		//FIXME: Too much sellout
		Description: "A language empowering everyone to build reliable and efficient software",
	},
	{
		URLName:      "sapper",
		DisplayName:  "Sapper",
		Author:       "Svelte",
		LearnMoreURL: "https://sapper.svelte.dev",
		Description:  "A <a href='https://svelte.dev'>Svelte</a>-powered framework for building web applications of all sizes, with a beautiful development experience and flexible filesystem-based routing",
	},
	{
		URLName:      "sass",
		DisplayName:  "SASS",
		LearnMoreURL: "https://sass-lang.com/",
		Description:  "“Syntactically Awesome Style Sheets”—A mature, stable, and powerful professional grade CSS extension language.",
	},
	{
		URLName:     "shell",
		DisplayName: "Shell",
	},
	{
		URLName:      "stylus",
		DisplayName:  "Stylus",
		LearnMoreURL: "https://stylus-lang.com",
		Description:  "Expressive, robust, feature-rich CSS language built for nodejs",
	},
	{
		URLName:      "svelte",
		DisplayName:  "Svelte",
		LearnMoreURL: "https://svelte.dev/",
		Description:  "A radical new approach to building user interfaces. Whereas traditional frameworks like React and Vue do the bulk of their work in the browser, Svelte shifts that work into a compile step that happens when you build your app.",
	},
	{
		URLName:      "toml",
		DisplayName:  "TOML",
		LearnMoreURL: "https://toml.io",
		Description:  "A minimal configuration file format that's easy to read due to obvious semantics. Designed to map unambiguously to a hash table.",
	},
	{
		URLName:      "typescript",
		DisplayName:  "TypeScript",
		Author:       "Microsoft",
		LearnMoreURL: "https://www.typescriptlang.org/",
		Description:  "An open-source language which builds on JavaScript by adding static type definitions",
	},
	{
		URLName:      "vue",
		DisplayName:  "Vue",
		Aliases:      []string{"vuejs"},
		LearnMoreURL: "https://vuejs.org",
		Description:  "The progressive JavaScript framework",
	},
	{
		URLName:      "webpack",
		DisplayName:  "Webpack",
		LearnMoreURL: "https://webpack.js.org/",
		Description:  "A bundler for javascript and friends. Packs many modules into a few bundled assets.",
	},
	{
		URLName:      "yaml",
		DisplayName:  "YAML",
		LearnMoreURL: "https://yaml.org/",
		Description:  "A human friendly data serialization standard for all programming languages",
	},
	{
		URLName:      "manim",
		DisplayName:  "Manim",
		Author:       "3Blue1Brown",
		LearnMoreURL: "https://www.manim.community/",
		Description:  "A community-maintained Python library for creating mathematical animations",
	},
	{
		URLName:      "lark",
		DisplayName:  "Lark",
		LearnMoreURL: "https://lark-parser.readthedocs.io/",
		Description:  "A modern parsing library for Python. Lark can parse any context-free grammar",
	},
}

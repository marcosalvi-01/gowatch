package ui

//go:generate npx tailwindcss --cwd ./../ -i ./ui/assets/input.css -o ./ui/static/css/output.css --minify

// Page title styling - reused across pages with responsive sizing
func titleClass() string {
	return "text-3xl sm:text-4xl lg:text-5xl font-bold mb-4 sm:mb-6 text-white text-center"
}

// Section title styling - reused multiple times with responsive sizing
func sectionTitleClass() string {
	return "text-xl sm:text-2xl font-semibold mb-3 sm:mb-4 text-white"
}

// Button styling - reused across components with responsive sizing
func btnClass() string {
	return "shadow-md px-4 sm:px-5 py-2 sm:py-2.5 my-2 sm:my-2.5 cursor-pointer bg-green-500 text-white border-none rounded-lg text-sm font-medium transition-all duration-200 hover:bg-green-600 hover:-translate-y-0.5 active:translate-y-0 min-w-0"
}

// Form input styling - reused for all inputs with responsive sizing
func formInputClass() string {
	return "px-3 py-2 border border-gray-300 rounded text-sm w-full max-w-full sm:max-w-none focus:outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-400 text-white"
}

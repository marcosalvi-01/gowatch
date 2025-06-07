package ui

//go:generate npx tailwindcss --cwd ./../ -i ./ui/assets/input.css -o ./ui/static/css/output.css

// Page title styling - reused across pages
func titleClass() string {
	return "text-4xl font-bold mb-2 text-white text-center"
}

// Section title styling - reused multiple times
func sectionTitleClass() string {
	return "text-2xl font-semibold mb-4 text-gray-800"
}

// Button styling - reused across components
func btnClass() string {
	return "shadow-md px-5 py-2.5 my-2.5 cursor-pointer bg-green-500 text-white border-none rounded-lg text-sm font-medium transition-all duration-200 hover:bg-green-600 hover:-translate-y-0.5 active:translate-y-0"
}

// Form input styling - reused for all inputs
func formInputClass() string {
	return "px-3 py-2 border border-gray-300 rounded text-sm w-72 max-w-full focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
}

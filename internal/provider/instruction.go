package provider

func fsInstruction() string {
	return "If the user requests filesystem changes (create/edit/delete/list/move/copy), call the fs tool. Do not claim changes without using fs."
}

package resource

import "github.com/valyala/fastrand"

type Sticker struct {
	Ok     bool
	FileId string
}

var Stickers = []Sticker{
	{Ok: true, FileId: "CAACAgIAAxkBAAIl1F_qpqBIsKDZkI_aB0f7oqUGIrMFAAL3AAP3AsgP0JfeP4rRPA4eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl1V_qptdW43fw0eSnUxAqqWmtfErwAAL5AAP3AsgP9zQfNR3ox5QeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl1l_qpuupBXGIyYCS5GrWefXKiTy0AALxAAMWQmsKfJbUUt2VvmIeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl11_qpvhmYXxrLyzfTfAGFxBHYTB4AAK4AAMWQmsKiQHJJr7C74EeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl2F_qpwWUILzNu8T5meOD_k5RWabGAAIwBQACP5XMCtv54PVdKBg1HgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl2V_qpxBXc3FmOZmBvyFKG6kqy115AAIyBQACP5XMCqAGktqw0HXQHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl2l_qpyBmdYMUXaLi9s6GEtSL_b-rAAJ_AAOWn4wODKHNuY8CLeceBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl21_qpy00Dqwm7bQlnXHhVGsYAxJOAAKLAAOWn4wOB__M_KeY00YeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl3F_qp0UfaCMw2zGYeXMv9x1677pTAALEBQACP5XMCrSn0eoopwziHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl3V_qp1fx4xElbCtWoO2wVG1K67cdAALFBQACP5XMCnvvCW__fY0OHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl3l_qp2aCDvhh75xehSh2OO36R_o6AALmAQACFkJrCkbP8M3aTyDhHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl4l_qp6HVzCF85j2bzPKZZQmd0r5xAALoAQACFkJrCgtMrejShZ9SHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIl41_qp63si1kfGfd76D1ncwj2KjmUAAKMAAOvxlEaHAiTF_b7ZUMeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIl5F_qp7uREs5TraUQMhJt40aH57wNAAKPAAOvxlEamlDt3-5_dXQeBA"},

	{Ok: false, FileId: "CAACAgIAAxkBAAImn1_qrTfopMIW8fLq6zkET7f38dVTAAJSAAOtZbwU2nZehNgFfdYeBA"},
	{Ok: true, FileId: "CAACAgIAAxkBAAImoF_qrVTcDmYxEExaMfOVLpmyjzG5AAJOAAOtZbwUIWzOXysr2zweBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImoV_qrWE7MIQKhWVqWTxxjtC_lWlYAAJfAAPBnGAMLpRna9tNe9QeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImol_qrW4DoA5PHCtW1E1ELgJjDg9QAAJaAAPBnGAMijYK3J9nNxceBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImo1_qrXzUlF4X6HQjL2mSBk5qkFS2AAK6AAP3AsgP_Qw4POqYV7keBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImpF_qrY5bPmkkUAU6P8nuBQvzvroYAAK8AAP3AsgPM6_Lz2J4moEeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImpV_qrZwd06bEPI1zSDqQKcp_JopTAAIbCQACGELuCNy5pdXzSq7IHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImpl_qraqoggABFW0T3176PdgkL8pT2AACGQkAAhhC7gjnA3EwBUHY7x4E"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImp1_qrbbyQMAVyXhR81ELEfOD-arPAAJeAAOmysgMnL25icai2v8eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImqF_qrcIow28Kw_32eX6St7mrnle1AAJfAAOmysgMIWV9tC1PxEceBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImqV_qreNVfyyvyiPpL5cNuQQAAYgk3gACgAMAAvPjvgvGNWb0eTUO5R4E"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImql_qre-M9o4gzHOCeJ_j2DgB7oc8AAJ_AwAC8-O-C-ndq7jpB6DQHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImq1_qrgoMzfVFLb8hwi_3tgEJ2uO0AAKnAAOtZbwU3zelGHR_Zz0eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImrF_qrh9mUm1qsHFfaYYbIPG7ePPAAAKmAAOtZbwUWxd-4t3CeYAeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImrV_qri4nv0O7DyqFZtZvjxPr8cQNAAL6AAPOGnYLfcq1syBEiQweBA"},
	{Ok: true, FileId: "CAACAgIAAxkBAAImrl_qrkLHIHPWXqGh466Gzn2KulvFAAL9AAPOGnYL1Bn6750mp4MeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImr1_qrnT3SmWVOgqLUnBLDzSddQABzQACugEAAjDUnREkSkV63sgHhB4E"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImsF_qroOsNstovn-Bnf__xReE8LK1AAKsAQACMNSdEewQN1Mzdgk-HgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImsV_qrpHg6hCXG0K5_iZrVW_iQoRUAAISAQAC5KDOB7xN3G05Qxk4HgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImsl_qrqnHxs_35CEbX2KIxPDSQgvsAAIPAQAC5KDOB3hCTpMPAxM_HgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIms1_qrrZDysNAWHf-XvzrP7D-CdhKAAKTAAPw2EUWdUyNGATgKo0eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImtF_qrskyuwz7eW0y_j4IfBSXIFTyAAKdAAPw2EUW8TIMKYUpZDEeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImtV_qruDTivL1RstxKIZRSyWPTi6eAAIHCwACLw_wBvVdW3Mb1FVjHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImtl_qru42Pf4Piahy91BYHUk4LAvmAAIOCwACLw_wBpEI4bLHoxydHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImt1_qrvrw4aQZettKjO1kZgvI4pYzAAIfAAMoD2oUimIMdxo7OPIeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImuF_qrwrVZBj6ITF3IbSdgXOR1iUMAAITAAMoD2oUlwYvsKF7R20eBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImuV_qryiAP43ulbOYpMKB_oK3wjpIAAIeAwACbbBCA6gI8-jQjopoHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImul_qrzncYQgCiu09Ygyo8TLhC8rHAAIWAwACbbBCA68qA0qDE3KBHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImu1_qr01NAAGtNyRqoTBuV8KKMsqFIwACRgADUomRI_j-5eQK1QodHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImvF_qr1sR2pB6qrDVQhs_V-IkDsP0AAI-AANSiZEjjHxD4k8_C24eBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImvV_qr2vM5dAuieWCf8mtdIZInaPwAAI7AQACFkJrCl-7BJlQlsKeHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImvl_qr30Cywa--JCUJxrxMceJPHihAAJEAQACFkJrCoHj1ZE-jFkxHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImv1_qr4wsZAbebrAvLdhItAEt140WAAKgAAP3AsgPw0cdAaCbwBoeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImwF_qr59m1o4feAX9osUZKMiWWwQtAAKZAAP3AsgPCsUxK8tzoGkeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAImwV_qr7FngA7C7QqyAAFqQENdY1QHUwACXgUAAj-VzAqq1ncTLO-MOR4E"},
	{Ok: false, FileId: "CAACAgIAAxkBAAImwl_qr74jinwKhyNlxZp7IcjKqTWYAAJhBQACP5XMCpwjR5fIeeKwHgQ"},

	{Ok: true, FileId: "CAACAgEAAxkBAAImzF_qr-V_CtfZ86v1fF_n-q1ffs5XAAIQAQACOA6CEf7Uv8SNQqXMHgQ"},
	{Ok: false, FileId: "CAACAgEAAxkBAAImzV_qr_xQ2nzLV1ll-W9XTpYbwCtmAAMBAAI4DoIRetPI_pltFLIeBA"},
}

func GenerateSticker(result bool) string {
	var available []Sticker
	for _, s := range Stickers {
		if s.Ok == result {
			available = append(available, s)
		}
	}

	idx := int(fastrand.Uint32n(uint32(len(available))))

	if len(available) > idx {
		return available[idx].FileId
	}

	return ""
}

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

	{Ok: true, FileId: "CAACAgIAAxkBAAIoaF_q-uKw8zZ-zdQTK0Ukr00WJ3KhAAIzBwACRvusBB9PEXZlMCInHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoaV_q-va81q7Kre_L9sW1AWgcMjWEAAI9BwACRvusBERQD397YIAjHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoal_q-wtoMlhtxM9p7koEXWwrQulXAAIuAQACIjeOBLAG8ITaL_y_HgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoa1_q-yBAL4gniyxHAZ9ZuHGAFLikAAImAQACIjeOBOOgmk7BxkpWHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIobF_q-y6M-AtGqS0hxNOCoBF2kjYtAAIjAANZu_wlkbzSYEBl88ceBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIobV_q-zyeklkMZ5IJkKpRoYf6bTsPAAIvAANZu_wldRiomYF4UW4eBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIobl_q-1K_Mitu-yKIsnjQvd1nN-ELAAJBAAN4qOYP-J7xorhFu34eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIob1_q-2Bvg5ZT5kLrKysVtq3BO5CaAAJGAAN4qOYPMeV85dgtrmgeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIocF_q-3Em2PLHzyvL6mkSypz_yFGOAAKiAQACFkJrCqF3d2OaToMhHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIocV_q-4A9JseA5y2Z22KDZyUrSJg4AALEAQACFkJrCoabSE_KkhbCHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIocl_q-42L0SVa9KGuY2nVKX59AnWPAAJmAANZu_wlnLyYkr-3hxceBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoc1_q-5sYGeGGNHSPLW0B3UANk5CvAAJUAANZu_wl69zrQijq35keBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIodF_q-7BsNW8nmugmyJ5yaJOpAr3_AAJTAAPANk8T_gJ3hXi7rlUeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIodV_q-8AICpiiIULNgzZiWMGFTLxVAAJbAAPANk8TQ-OZK7NCh4oeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIodl_q-9pWDw3La6KZ2FWm12LFc4UvAAJWAANBtVYM1ZNlBU-tNFYeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIod1_q--OxDyLayRUdhDOVY_DS8udnAAJYAANBtVYMmYb-WPoiBC8eBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoeF_q-_Q48j5fxtZbdW1k-Cw7UKE5AAJqAAOmysgMa0ruxmIoCS0eBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoeV_q_AIc2cQ77q7pUhaaxAuxmmWWAAJxAAOmysgMI--w3MfRamQeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoel_q_BODGmUdEdFqqLoqO_iNX-sgAAL1AAMw1J0R3NeLwV6aUvUeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoe1_q_CVr8g4mDTSUD67XWK1f9vucAAICAQACMNSdEQeri5WwHbRSHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIofF_q_DHID79PuQL1Emb4D3Jilz3VAAKnAAMWQmsK2XM0SpQ0cykeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIofV_q_EHznp8zxDOUqVpGWrs5KkndAAKgAAMWQmsKBAICCOb4TIYeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIofl_q_FPAyUeGYCbKN1thiPtCEaBtAAJCAANZu_wl3sFvoDs_W9EeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIof1_q_GIS3j15ilBuag7eUPHLjARvAAJBAANZu_wlHXGYEay5UmkeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIogF_q_G3yeBNJTisd06Dhg3GxI7tyAAIUBwACRvusBO7y0rmZcxR1HgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIogV_q_HzdRXthjZLxx2GTQL-ijRKnAAIYBwACRvusBBMDPSb7UQomHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIogl_q_Iq9hUFUAa39K2fjzD0GAr5zAAIiAAOQ_ZoVYR1wt7ToffoeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIog1_q_JirnpOhMJA9Sw-9VnFalqC8AAImAAOQ_ZoV90DVcZ0iz3MeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIohF_q_KTvR8MIzb9CZyAHu9Vqm1LyAAJqAAPANk8T_puXe-wcB9oeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIohV_q_LYccTsiM34Pj_o_Ocb76dCLAAJ2AAPANk8TdBqmNBsYSckeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIohl_q_MLCSXxcnz7O5u335o0ON5RBAAJEAwACtXHaBgy3OxO0hkMwHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoh1_q_NFOxdC3D3MWFV9HiGKPaQiPAAJQAwACtXHaBsOq9o3QxaLKHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoiF_q_OHMi2aS1SjTBCNSp3eCleFNAAJLAQACMNSdEQGFhnc2xMjpHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoiV_q_O5r_CVxK419yX9Be8nFQGtBAAI3AQACMNSdEcordJB3YvwBHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoil_q_Pg_SOId7_FFkQQM7Szw_YW0AAIbAAMkcWIafZSybkF0MBweBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoi1_q_QVuFojYWYBswo90hx1UuF70AAIqAAMkcWIaS9XVZNqIapweBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIojF_q_RLo4nJzvTw7L22PKfEMUJiqAAJaAANEDc8Xd6GRVdGwNkseBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIojV_q_RwlUnKZ01LV_zrLwEL2mEDnAAJfAANEDc8XV89L68IAAY8xHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIojl_q_S8DcZkPqfll_bd9SvP9IFu4AAIaAAP3AsgPry8JaXKONsseBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIoj1_q_TrTNKDdQI8VMz1p9p4aYo1fAAIbAAP3AsgPySwFywkZsl4eBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIokF_q_Uv31NDccArHF7ekxDd1Yt_kAAJAAAOvxlEaV1XfcKI2zaoeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIokV_q_VjrAAGOktQjXI74XWRbWK2udwACPQADr8ZRGk249Sh3NPS6HgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIokl_q_WoeaegRpSOvAsDdnc70jS1QAAIPAgACVp29CoJWy3lpHf-0HgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIok1_q_X1PdOZERXgSHlBoNB4aEia5AAIQAgACVp29Cs8tsLs99KMpHgQ-0HgQ"},

	{Ok: false, FileId: "CAACAgIAAxkBAAIolF_q_ZNV301KkptfCtehqMpZaLvsAAIZAAOvxlEaVREirkTNozAeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIolV_q_aYIR-jGQVY77c_r6KXxoFo0AAImAAOvxlEaD-pWA63QKMUeBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIoll_q_b2Bt-Aqqebgx3WORCqQq9yQAAIpAAPBnGAM8EupHr_Y33weBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIol1_q_dDLApQTuh25iA3V3PZe6w8BAAImAAPBnGAMnxWpvJx5YPEeBA"},

	{Ok: false, FileId: "CAACAgIAAxkBAAIomF_q_ds-LjpHE2-WDdfzl88VvMYNAAItAANEDc8XMuoxGuoTph4eBA"},
	{Ok: true, FileId: "CAACAgIAAxkBAAIomV_q_eN4QJP_Wj4nSvF0AcpxpgtgAAIsAANEDc8XyPvc7VqXDYseBA"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIom1_q_f_OgabIvUKvF1upDGqgDhsnAALoAgACtXHaBlINrrZVgIJbHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIonF_q_gtuGEvZ6WnKYaAoaLw6x4VGAALpAgACtXHaBh48aayPD7m-HgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIonV_q_ijHAYLWwFEE_5f7Lbv3lb4oAAJyAAM7YCQUiaK3KHtkgiceBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIonl_q_jgYUhbSlrD7uvivX0nz2ljPAAJzAAM7YCQUulEM8bKtM1IeBA"},

	{Ok: false, FileId: "CAACAgIAAxkBAAIooV_q_mKON5aI4MH8yGph4NZXcroLAAIsAwACusCVBS7Ioyk1lczrHgQ"},
	{Ok: true, FileId: "CAACAgIAAxkBAAIoo1_q_nF70p2gzGooFpqnh9Wbwia2AAIlAwACusCVBbpVJXwAAdR17x4E"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIopF_q_oeN3PIEQ59Mu2ANEcRwGdnnAAJTBAACzFRJCfnVQcY3NDMPHgQ"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIopV_q_pRjuUmay_d7T-8n8tZc627rAAJcBAACzFRJCTCAN0HcxE9eHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIopl_q_q27y-oqpGqduJ0xibNhOZznAAK0AAOWn4wOP4VP3hlFLAgeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIop1_q_rwsraYsRJ9WY3IKNW69TwgAA64AA5afjA55njYvvWKC5R4E"},

	{Ok: true, FileId: "CAACAgEAAxkBAAIoqF_q_tAvfSB6-ThenUKUtr39Q2IWAAL7CQACv4yQBAwndSWTBrDuHgQ"},
	{Ok: false, FileId: "CAACAgEAAxkBAAIoqV_q_t7mG3cFRUMRvJIiHcGmVAWSAAIMCgACv4yQBO719xB-FttLHgQ"},

	{Ok: false, FileId: "CAACAgIAAxkBAAIoq1_q_vXLzC8wOWHGDGnsb4_TXFb4AAILCQACGELuCEGTJRyJr2_IHgQ"},
	{Ok: true, FileId: "CAACAgIAAxkBAAIorF_q_wJT7TzPqC_xMgtBQDCsZi17AAL_CAACGELuCGIVP_VS1l7PHgQ"},

	{Ok: true, FileId: "CAACAgIAAxkBAAIorV_q_xQJqoucwc4FEMNdfg4kbaTHAAJFAAPBnGAMrUe7kkil4IkeBA"},
	{Ok: false, FileId: "CAACAgIAAxkBAAIorl_q_yBZjrCpEZIeQd5_tnYiB2_rAAI3AAPBnGAMiMUzXjSctCMeBA"},

	{Ok: false, FileId: "CAACAgIAAxkBAAIor1_q_3M8dedFBasqO5lwr_aPkprPAAJEAQAClp-MDjJQAAE26l5EgR4E"},
	{Ok: true, FileId: "CAACAgIAAxkBAAIosF_q_3tpPbJSG96LUY_FsuyzmmGRAAJPAQAClp-MDuqKyOWfpfQaHgQ"},
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

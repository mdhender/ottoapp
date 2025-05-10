// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import "net/http"

func (s *Server) routes() *http.ServeMux {
	s.mux = http.NewServeMux()

	//s.mux.HandleFunc("GET /login/{clan_id}/{magic_link}", s.getLoginClanIdMagicLink())

	s.mux.HandleFunc("GET /about", s.getHeroPage(s.paths.components, "about"))
	s.mux.HandleFunc("GET /calendar", s.getCalendar(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /contact-us", s.getHeroPage(s.paths.components, "contact-us"))
	s.mux.HandleFunc("GET /dashboard", s.getDashboard(s.paths.components, s.blocks.Footer, s.features.cacheBuster))
	s.mux.HandleFunc("GET /docs", s.getHeroPage(s.paths.components, "docs"))
	s.mux.HandleFunc("GET /docs/converting-turn-reports", s.getHeroPage(s.paths.components, "docs/converting-turn-reports"))
	s.mux.HandleFunc("GET /docs/dashboard-overview", s.getHeroPage(s.paths.components, "docs/dashboard-overview"))
	s.mux.HandleFunc("GET /docs/errors", s.getHeroPage(s.paths.components, "docs/errors"))
	s.mux.HandleFunc("GET /docs/getting-started", s.getHeroPage(s.paths.components, "docs/getting-started"))
	s.mux.HandleFunc("GET /docs/map-key", s.getHeroPage(s.paths.components, "docs/map-key"))
	s.mux.HandleFunc("GET /docs/ottomap-for-tribenet", s.getHeroPage(s.paths.components, "docs/ottomap-for-tribenet"))
	s.mux.HandleFunc("GET /docs/report-layout", s.getHeroPage(s.paths.components, "docs/report-layout"))
	s.mux.HandleFunc("GET /get-started", s.getHeroPage(s.paths.components, "get-started"))
	s.mux.HandleFunc("GET /learn-more", s.getHeroPage(s.paths.components, "learn-more"))
	s.mux.HandleFunc("GET /privacy", s.getHeroPage(s.paths.components, "privacy"))
	s.mux.HandleFunc("GET /settings", s.getSettings(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /settings/general", s.getSettingsGeneral(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /settings/general/timezone", s.getSettingsGeneralTimezone(s.paths.components))
	s.mux.HandleFunc("POST /settings/general/timezone", s.postSettingsGeneralTimezone(s.paths.components))
	s.mux.HandleFunc("GET /settings/notifications", s.getSettingsGeneral(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /settings/plans", s.getSettingsPlans(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /settings/security", s.getSettingsGeneral(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /trusted", s.getHeroPage(s.paths.components, "trusted"))

	s.mux.HandleFunc("GET /login", s.getLogin(s.paths.components))
	s.mux.HandleFunc("GET /login/clan/{clan_id}", s.getLoginClanId(s.paths.components))
	s.mux.HandleFunc("POST /login/clan/{clan_id}", s.postLoginClanId())
	s.mux.HandleFunc("GET /logout", s.getLogout())

	s.mux.HandleFunc("DELETE /errlog/{log_id}", s.deleteErrorLogLogId(s.paths.components))
	s.mux.HandleFunc("GET /errlog/{log_id}", s.getErrorLogLogId())

	s.mux.HandleFunc("DELETE /log/{log_id}", s.deleteLogLogId(s.paths.components))
	s.mux.HandleFunc("GET /log/{log_id}", s.getLogLogId())

	s.mux.HandleFunc("GET /maps", s.getMaps(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("DELETE /map/{map_id}", s.deleteMapMapId(s.paths.components))
	s.mux.HandleFunc("GET /map/{map_id}", s.getMapMapId())

	s.mux.HandleFunc("GET /reports", s.getReports(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("DELETE /report/{report_id}", s.deleteReportReportId(s.paths.components))
	s.mux.HandleFunc("GET /report/{report_id}", s.getReportReportId())
	s.mux.HandleFunc("GET /report/beta/docx-to-json", s.getReportBetaDocxToJson())
	s.mux.HandleFunc("GET /report/beta/docx-to-text", s.getReportBetaDocxToText())
	s.mux.HandleFunc("GET /reports/uploads", s.getReportsUploads(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /reports/uploads/failed", s.getReportsUploadsFailed(s.paths.components))
	s.mux.HandleFunc("GET /reports/uploads/plain-text", s.getReportsUploadsPlainText(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("GET /reports/uploads/success", s.getReportsUploadsSuccess(s.paths.components))
	s.mux.HandleFunc("GET /reports/turn/{turn_id}/clan/{clan_id}", s.getReportsTurnIdClanId(s.paths.components))

	s.mux.HandleFunc("GET /reports/dropbox/upload", s.getReportsDropboxUpload(s.paths.components, s.blocks.Footer))
	s.mux.HandleFunc("POST /reports/dropbox/scrub", s.postDropboxScrub(s.paths.components, version.String()))
	s.mux.HandleFunc("POST /reports/dropbox/upload", s.postDropboxUpload(s.paths.components))

	s.mux.HandleFunc("POST /reports/plain-text/scrub", s.postPlainTextScrub(s.paths.components, s.paths.userdata))
	s.mux.HandleFunc("POST /reports/plain-text/upload", s.postPlainTextUpload(s.paths.components, s.blocks.Footer))

	//s.mux.HandleFunc("GET /reports/docx/upload", s.getReportsDocxUpload(s.paths.components, s.blocks.Footer))
	//s.mux.HandleFunc("POST /reports/docx/upload", s.postDocxUpload(s.paths.components))
	//s.mux.HandleFunc("GET /reports/uploads/msword", func(w http.ResponseWriter, r *http.Request) {
	//	http.Redirect(w, r, "/reports/docx/upload", http.StatusSeeOther)
	//})
	////s.mux.HandleFunc("POST /reports/uploads/msword", s.postReportsUploadsMSWord(s.paths.components, s.blocks.Footer))

	s.mux.HandleFunc("GET /api/v1/paths", s.getApiPathsV1())
	s.mux.HandleFunc("GET /api/v1/version", s.getApiVersionV1())
	s.mux.HandleFunc("GET /api/v1/clan-files/{clan_id}", s.getApiClanFilesV1(s.paths.userdata))
	s.mux.HandleFunc("POST /api/v1/report/upload/docx", s.postApiReportUploadDocx(s.paths.userdata))
	s.mux.HandleFunc("POST /api/v1/report/upload/file", s.postApiReportUploadFile(s.paths.userdata))
	s.mux.HandleFunc("POST /api/v1/report/upload/text", s.postApiReportUploadText(s.paths.userdata))

	// unfortunately for us, the "/" route is special. it serves the landing page as well as all the assets.
	//s.mux.Handle("GET /", http.FileServer(http.Dir(s.paths.assets)))
	s.mux.Handle("GET /", s.getIndex(s.staticFileServer, s.paths.assets, s.getHeroPage(s.paths.components, "landing")))

	return s.mux
}

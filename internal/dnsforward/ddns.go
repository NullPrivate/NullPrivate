package dnsforward

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"text/template"

	"github.com/AdGuardPrivate/AdGuardPrivate/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

//go:embed ddns-template/*
var ddnsTemplates embed.FS

// Template file path constants
const (
	ddnsTemplateDirPath = "ddns-template"
	unixTemplateFile    = "unix.sh"
	windowsTemplateFile = "windows.ps1"
	macosTemplateFile   = "unix.sh" // macOS shares template with Unix
)

// Script download Content-Type
const (
	contentTypeBash       = "application/x-sh"
	contentTypePowerShell = "application/octet-stream"
)

// User-friendly error messages
const (
	errMsgDDNSGeneric    = "Unable to generate DDNS script, please try again later"
	errMsgDDNSNoTemplate = "DDNS script template not found"
	errMsgDDNSNoDomain   = "Unable to determine domain information"
)

// Template data structure - using uppercase field names (Go convention)
type ddnsTemplateData struct {
	ServerName string
	Username   string
	Password   string
	Domain     string
	Cookies    string
}

// Register DDNS script download handlers
func (s *Server) registerDDNSHandlers() {
	if s.conf.HTTPRegister == nil {
		return
	}

	s.conf.HTTPRegister(http.MethodGet, "/control/ddns/script/windows", s.handleDDNSWindowsScript)
	s.conf.HTTPRegister(http.MethodGet, "/control/ddns/script/linux", s.handleDDNSLinuxScript)
	s.conf.HTTPRegister(http.MethodGet, "/control/ddns/script/macos", s.handleDDNSMacOSScript)
}

// Handle Windows script request
func (s *Server) handleDDNSWindowsScript(w http.ResponseWriter, r *http.Request) {
	s.handleDDNSScript(w, r, windowsTemplateFile, "ddns-script.ps1", contentTypePowerShell)
}

// Handle Linux script request
func (s *Server) handleDDNSLinuxScript(w http.ResponseWriter, r *http.Request) {
	s.handleDDNSScript(w, r, unixTemplateFile, "ddns-script.sh", contentTypeBash)
}

// Handle macOS script request
func (s *Server) handleDDNSMacOSScript(w http.ResponseWriter, r *http.Request) {
	s.handleDDNSScript(w, r, macosTemplateFile, "ddns-script.sh", contentTypeBash)
}

// handleDDNSScript is a generic method for handling DDNS script download
func (s *Server) handleDDNSScript(w http.ResponseWriter, r *http.Request, templateFileName, downloadFileName, contentType string) {
	// Get domain from URL parameter first, fallback to server name if not provided
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		domain = "nas.home"
		log.Debug("Using server name as DDNS domain: %s", domain)
	} else {
		log.Debug("Using provided DDNS domain: %s", domain)
	}

	// Get server address (including protocol prefix)
	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Use X-Forwarded-Proto header if present
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	server := fmt.Sprintf("%s://%s", scheme, host)

	// Get cookie
	var cookieStr string
	cookies := r.Cookies()
	var cookieParts []string
	for _, cookie := range cookies {
		if cookie.Name == "agh_session" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}
	cookieStr = strings.Join(cookieParts, "; ")

	// Log
	if cookieStr != "" {
		log.Debug("DDNS: Found cookie: %s", cookieStr)
	} else {
		log.Debug("DDNS: agh_session cookie not found")
	}

	// Prepare template data - using uppercase field names
	data := ddnsTemplateData{
		ServerName: server,
		Username:   "", // User will fill in themselves
		Password:   "", // User will fill in themselves
		Domain:     domain,
		Cookies:    cookieStr, // Add obtained cookie
	}

	// Read template file
	templatePath := fmt.Sprintf("%s/%s", ddnsTemplateDirPath, templateFileName)
	tmplContent, err := fs.ReadFile(ddnsTemplates, templatePath)
	if err != nil {
		log.Error("Failed to read DDNS script template: %v", err)
		aghhttp.Error(r, w, http.StatusInternalServerError, errMsgDDNSNoTemplate)
		return
	}

	// Create template function map
	funcMap := template.FuncMap{
		// Map lowercase variable names in template to uppercase field names in struct
		"server_name": func() string { return data.ServerName },
		"username":    func() string { return data.Username },
		"password":    func() string { return data.Password },
		"domain":      func() string { return data.Domain },
		"cookies":     func() string { return data.Cookies },
	}

	// Parse template using function map
	tmpl, err := template.New("ddns_script").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		log.Error("Failed to parse DDNS script template: %v", err)
		aghhttp.Error(r, w, http.StatusInternalServerError, errMsgDDNSGeneric)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadFileName))

	// Render template to response
	err = tmpl.Execute(w, nil) // Using nil as data since we use function map
	if err != nil {
		log.Error("Failed to generate DDNS script: %v", err)
		// Cannot send error since we've already started writing the response
		return
	}
}

// Initialize DDNS handlers
func (s *Server) initDDNS() {
	s.registerDDNSHandlers()
}

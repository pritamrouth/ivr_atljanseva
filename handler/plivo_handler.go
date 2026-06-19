package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ivr_ataljanseva/db/repository"
	"ivr_ataljanseva/models"
	"ivr_ataljanseva/plivo"
)

type PlivoHandler struct {
	citizenRepo   *repository.CitizenRepository
	politicalRepo *repository.PoliticalUserRepository
	baseURL       string
	audioBaseURL  string
	maxRetries    int
}

func NewPlivoHandler(
	citizenRepo *repository.CitizenRepository,
	politicalRepo *repository.PoliticalUserRepository,
	baseURL, audioBaseURL string,
) *PlivoHandler {
	return &PlivoHandler{
		citizenRepo:   citizenRepo,
		politicalRepo: politicalRepo,
		baseURL:       baseURL,
		audioBaseURL:  audioBaseURL,
		maxRetries:    3,
	}
}

// --------------- step audio file names per language ---------------

var stepAudioFiles = map[string]map[string]string{
	"english": {
		"welcome":      "Step_1.mp3",
		"ward_input":   "Step 2 – User Identification.mp3",
		"sos_menu":     "Step 3 – Emergency Services.mp3",
		"record_start": "Step 5 – Record Complaint.mp3",
		"confirmation": "Step 6 – Complaint Confirmation.mp3",
	},
	"hindi": {
		"ward_input":   "चरण 2 – उपयोगकर्ता पहचान.mp3",
		"sos_menu":     "चरण 3 – आपातकालीन सेवाएँ.mp3",
		"record_start": "चरण 5 – शिकायत दर्ज करें.mp3",
		"confirmation": "चरण 6 – शिकायत की पुष्टि.mp3",
	},
	"marathi": {
		"ward_input":   "पायरी २ – वापरकर्ता ओळख.mp3",
		"sos_menu":     "पायरी ३ – आपत्कालीन सेवा.mp3",
		"record_start": "पायरी ५ – तक्रार नोंदवा.mp3",
		"confirmation": "पायरी ६ – तक्रार पुष्टी.mp3",
	},
}

func (h *PlivoHandler) playURL(step, lang string) string {
	if h.audioBaseURL == "" {
		return ""
	}
	files, ok := stepAudioFiles[lang]
	if !ok {
		return ""
	}
	file, ok := files[step]
	if !ok {
		return ""
	}
	return plivo.AudioURL(h.audioBaseURL, lang, file)
}

// --------------- step 0: incoming call ---------------

// POST /ivr/plivo/incoming
func (h *PlivoHandler) Incoming(c *gin.Context) {
	phone := c.PostForm("From")
	if phone == "" {
		c.String(http.StatusBadRequest, plivo.Response(
			plivo.Speak("Invalid request. No caller ID.", "english"),
			plivo.Hangup(),
		))
		return
	}

	phone = normalizePhone(phone)

	citizen, err := h.citizenRepo.FindByPhone(c.Request.Context(), phone)
	if err != nil {
		log.Printf("citizen lookup error: %v", err)
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("System error. Please try again later.", "english"),
			plivo.Hangup(),
		))
		return
	}

	if citizen != nil {
		lang := citizen.Language
		nsName := ""
		nsPhone := ""
		if citizen.NagarsevakID != uuid.Nil {
			ns, err := h.politicalRepo.FindNagarsevakByID(c.Request.Context(), citizen.NagarsevakID)
			if err == nil && ns != nil {
				nsName = ns.Name
				nsPhone = ns.Phone
			}
		}
		h.returnMainMenu(c, phone, lang, citizen.Pincode, citizen.Ward, citizen.NagarsevakID.String(), nsName, nsPhone)
		return
	}

	// new user – language selection
	action := h.baseURL + "/ivr/plivo/language?phone=" + phone
	url := h.playURL("welcome", "english")
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigits(action, 1, 10, plivo.Play(url)),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigits(action, 1, 10, plivo.Speak("Welcome to Atal Janseva Citizen Service. Marathi saaathi 1 daba. For English, press 2. Hindi ke liye 3 dabaie.", "english")),
		))
	}
}

// --------------- step 1: language selection ---------------

// POST /ivr/plivo/language
func (h *PlivoHandler) LanguageSelect(c *gin.Context) {
	phone := c.Query("phone")
	digits := c.PostForm("Digits")

	lang := resolveLanguage(digits)

	action := h.baseURL + "/ivr/plivo/pincode-input?phone=" + phone + "&language=" + lang

	url := h.playURL("ward_input", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigitsEx(action, 12, 15, "", 4, plivo.Play(url)),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigitsEx(action, 12, 15, "", 4, plivo.Speak("Enter your 6-digit pincode, then press hash, then your ward number.", lang)),
		))
	}
}

// --------------- step 1b: pincode + ward input (combined "400601#21" format) ---------------

// POST /ivr/plivo/pincode-input
func (h *PlivoHandler) PincodeInput(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	digits := c.PostForm("Digits")

	parts := strings.SplitN(digits, "#", 2)
	pincode := strings.TrimSpace(parts[0])

	if pincode == "" || len(pincode) < 6 {
		action := h.baseURL + "/ivr/plivo/pincode-input?phone=" + phone + "&language=" + lang
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigitsEx(action, 12, 15, "", 4,
				plivo.Speak("Invalid input. Enter your 6-digit pincode, hash, then ward number.", lang),
			),
		))
		return
	}

	pincode = pincode[:6]

	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		wardInput := strings.TrimSpace(parts[1])
		h.resolveWardAndProceed(c, phone, lang, pincode, wardInput, 0)
		return
	}

	action := h.baseURL + "/ivr/plivo/ward-input?phone=" + phone + "&language=" + lang + "&pincode=" + pincode
	c.String(http.StatusOK, plivo.Response(
		plivo.GetDigits(action, 10, 15, plivo.Speak("Now enter your ward number and press hash.", lang)),
	))
}

// --------------- step 2: ward input (pincode from query, ward from digits) ---------------

// POST /ivr/plivo/ward-input
func (h *PlivoHandler) WardInput(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	wardInput := c.PostForm("Digits")
	retryStr := c.Query("retry")

	if wardInput == "" {
		h.wardInputRetry(c, phone, lang, pincode, retryStr)
		return
	}

	retry, _ := strconv.Atoi(retryStr)
	h.resolveWardAndProceed(c, phone, lang, pincode, wardInput, retry)
}

// --------------- step 2b: ward select from list ---------------

// POST /ivr/plivo/ward-select
func (h *PlivoHandler) WardSelect(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	wardsRaw := c.Query("wards")
	digits := c.PostForm("Digits")

	idx, _ := strconv.Atoi(digits)
	idx--

	wards := strings.Split(wardsRaw, ",")
	if idx < 0 || idx >= len(wards) {
		action := h.baseURL + "/ivr/plivo/ward-input?phone=" + phone + "&language=" + lang + "&pincode=" + pincode + "&retry="
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("Invalid selection. Please try again.", lang),
			plivo.GetDigits(action, 10, 15),
		))
		return
	}

	selectedWard := strings.TrimSpace(wards[idx])

	nagarsevaks, err := h.politicalRepo.FindNagarsevaks(c.Request.Context(), pincode, selectedWard)
	if err != nil {
		log.Printf("nagarsevak lookup error: %v", err)
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("System error. Please try again later.", lang),
			plivo.Hangup(),
		))
		return
	}

	switch {
	case len(nagarsevaks) == 0:
		h.returnWhatsAppPrompt(c, lang)

	case len(nagarsevaks) == 1:
		ns := nagarsevaks[0]
		err := h.citizenRepo.UpsertCitizen(c.Request.Context(), phone, lang, pincode, selectedWard, ns.ID)
		if err != nil {
			log.Printf("auto-save error: %v", err)
		}
		h.returnMainMenu(c, phone, lang, pincode, selectedWard, ns.ID.String(), ns.Name, ns.Phone)

	case len(nagarsevaks) <= 5:
		h.returnNagarsevakMenu(c, phone, lang, pincode, selectedWard, nagarsevaks)

	default:
		h.returnWhatsAppPrompt(c, lang)
	}
}

// --------------- step 2c: nagarsevak select from list ---------------

// POST /ivr/plivo/nagarsevak-select
func (h *PlivoHandler) NagarsevakSelect(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	ward := c.Query("ward")
	idsRaw := c.Query("ids")
	digits := c.PostForm("Digits")

	idx, _ := strconv.Atoi(digits)
	idx--

	ids := strings.Split(idsRaw, ",")
	if idx < 0 || idx >= len(ids) {
		action := h.baseURL + "/ivr/plivo/ward-select?phone=" + phone + "&language=" + lang + "&pincode=" + pincode + "&wards=" + ward
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("Invalid selection. Please try again.", lang),
			plivo.GetDigits(action, 1, 10),
		))
		return
	}

	nsID := strings.TrimSpace(ids[idx])
	parsedUUID, err := uuid.Parse(nsID)
	if err != nil {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("System error. Please try again later.", lang),
			plivo.Hangup(),
		))
		return
	}

	ns, err := h.politicalRepo.FindNagarsevakByID(c.Request.Context(), parsedUUID)
	if err != nil || ns == nil {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("System error. Please try again later.", lang),
			plivo.Hangup(),
		))
		return
	}

	err = h.citizenRepo.UpsertCitizen(c.Request.Context(), phone, lang, pincode, ward, ns.ID)
	if err != nil {
		log.Printf("save error: %v", err)
	}

	h.returnMainMenu(c, phone, lang, pincode, ward, ns.ID.String(), ns.Name, ns.Phone)
}

// --------------- step 3: unified SOS / main menu (same for everyone) ---------------

// POST /ivr/plivo/main-menu
func (h *PlivoHandler) MainMenu(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	ward := c.Query("ward")
	nsID := c.Query("nagarsevak_id")
	nsName := c.Query("nagarsevak_name")
	nsPhone := c.Query("nagarsevak_phone")
	digits := c.PostForm("Digits")

	switch digits {
	case "1":
		h.sosConnect(c, lang, "Fire Emergency")
	case "2":
		h.sosConnect(c, lang, "Medical Emergency")
	case "3":
		h.corporatorConnect(c, phone, lang, pincode, ward, nsID, nsName, nsPhone)
	default:
		h.returnMainMenu(c, phone, lang, pincode, ward, nsID, nsName, nsPhone)
	}
}

// --------------- step 3b: complaint recording ---------------

// POST /ivr/plivo/complaint-record
func (h *PlivoHandler) ComplaintRecord(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	ward := c.Query("ward")
	nsID := c.Query("nagarsevak_id")
	nsName := c.Query("nagarsevak_name")
	nsPhone := c.Query("nagarsevak_phone")

	action := h.baseURL + "/ivr/plivo/complaint-callback?phone=" + phone +
		"&language=" + lang +
		"&pincode=" + pincode +
		"&ward=" + ward +
		"&nagarsevak_id=" + nsID +
		"&nagarsevak_name=" + nsName +
		"&nagarsevak_phone=" + nsPhone

	url := h.playURL("record_start", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.Play(url),
			plivo.RecordWithBeep(action, 120, "#"),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("After the beep, please describe your complaint and mention the address or nearby landmark. Press hash when done.", lang),
			plivo.RecordWithBeep(action, 120, "#"),
		))
	}
}

// POST /ivr/plivo/complaint-callback
func (h *PlivoHandler) ComplaintCallback(c *gin.Context) {
	lang := c.Query("language")
	nsName := c.Query("nagarsevak_name")
	// recordUrl is sent by Plivo in the POST body
	_ = c.PostForm("RecordUrl")
	_ = c.PostForm("RecordingDuration")

	if nsName == "" {
		nsName = "your nagarsevak"
	}

	url := h.playURL("confirmation", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.Play(url),
			plivo.Hangup(),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("Your complaint has been successfully registered. Our team will process your request shortly. Thank you.", lang),
			plivo.Hangup(),
		))
	}
}

// --------------- dial status callback ---------------

// POST /ivr/plivo/dial-status
func (h *PlivoHandler) DialStatus(c *gin.Context) {
	phone := c.Query("phone")
	lang := c.Query("language")
	pincode := c.Query("pincode")
	ward := c.Query("ward")
	nsID := c.Query("nagarsevak_id")
	nsName := c.Query("nagarsevak_name")
	nsPhone := c.Query("nagarsevak_phone")

	dialStatus := c.PostForm("DialStatus")

	if dialStatus != "completed" {
		h.returnComplaintOffer(c, phone, lang, pincode, ward, nsID, nsName, nsPhone)
		return
	}

	c.String(http.StatusOK, plivo.Response(
		plivo.Speak("Thank you for using Atal Janseva. Goodbye.", lang),
		plivo.Hangup(),
	))
}

func (h *PlivoHandler) returnComplaintOffer(c *gin.Context, phone, lang, pincode, ward, nsID, nsName, nsPhone string) {
	action := h.baseURL + "/ivr/plivo/complaint-record?phone=" + phone +
		"&language=" + lang +
		"&pincode=" + pincode +
		"&ward=" + ward +
		"&nagarsevak_id=" + nsID +
		"&nagarsevak_name=" + nsName +
		"&nagarsevak_phone=" + nsPhone

	c.String(http.StatusOK, plivo.Response(
		plivo.Speak("The office is currently unavailable. Your complaint will be recorded.", lang),
		plivo.GetDigits(action, 1, 10, plivo.Speak("To register a complaint, press 9.", lang)),
	))
}

// --------------- internal response builders ---------------

func (h *PlivoHandler) resolveWardAndProceed(c *gin.Context, phone, lang, pincode, wardInput string, retry int) {
	matches, err := h.politicalRepo.FindMatchingWards(c.Request.Context(), pincode, wardInput)
	if err != nil {
		log.Printf("ward resolve error: %v", err)
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("System error. Please try again later.", lang),
			plivo.Hangup(),
		))
		return
	}

	switch {
	case len(matches) == 0:
		h.wardInputRetry(c, phone, lang, pincode, strconv.Itoa(retry+1))

	case len(matches) == 1:
		selectedWard := matches[0].Ward
		nagarsevaks, err := h.politicalRepo.FindNagarsevaks(c.Request.Context(), pincode, selectedWard)
		if err != nil {
			log.Printf("nagarsevak lookup error: %v", err)
			c.String(http.StatusOK, plivo.Response(
				plivo.Speak("System error. Please try again later.", lang),
				plivo.Hangup(),
			))
			return
		}
		switch {
		case len(nagarsevaks) == 0:
			h.returnWhatsAppPrompt(c, lang)
		case len(nagarsevaks) == 1:
			ns := nagarsevaks[0]
			err := h.citizenRepo.UpsertCitizen(c.Request.Context(), phone, lang, pincode, selectedWard, ns.ID)
			if err != nil {
				log.Printf("auto-save error: %v", err)
			}
			h.returnMainMenu(c, phone, lang, pincode, selectedWard, ns.ID.String(), ns.Name, ns.Phone)
		case len(nagarsevaks) <= 5:
			h.returnNagarsevakMenu(c, phone, lang, pincode, selectedWard, nagarsevaks)
		default:
			h.returnWhatsAppPrompt(c, lang)
		}

	case len(matches) <= 4:
		h.returnWardMenu(c, phone, lang, pincode, matches)

	default:
		h.returnWhatsAppPrompt(c, lang)
	}
}

func (h *PlivoHandler) returnMainMenu(c *gin.Context, phone, lang, pincode, ward, nsID, nsName, nsPhone string) {
	if nsName == "" {
		nsName = "your nagarsevak"
	}

	action := h.baseURL + "/ivr/plivo/main-menu?phone=" + phone + "&language=" + lang +
		"&pincode=" + pincode + "&ward=" + ward +
		"&nagarsevak_id=" + nsID + "&nagarsevak_name=" + nsName +
		"&nagarsevak_phone=" + nsPhone

	url := h.playURL("sos_menu", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("NAGARSEVAK "+nsName+" IS CONNECTED.", lang),
			plivo.GetDigits(action, 1, 10, plivo.Play(url)),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("NAGARSEVAK "+nsName+" IS CONNECTED.", lang),
			plivo.GetDigits(action, 1, 10, plivo.Speak("For Emergency SOS Assistance: Press 1 for Fire Emergency. Press 2 for Medical or Accident Emergency. Press 3 to connect with your Corporator.", lang)),
		))
	}
}

func (h *PlivoHandler) sosConnect(c *gin.Context, lang, emergencyType string) {
	c.String(http.StatusOK, plivo.Response(
		plivo.Speak("Connecting you to "+emergencyType+". Please hold.", lang),
		plivo.Hangup(),
	))
}

func (h *PlivoHandler) complaintRecordXML(c *gin.Context, phone, lang, pincode, ward, nsID, nsName, nsPhone string) {
	action := h.baseURL + "/ivr/plivo/complaint-callback?phone=" + phone +
		"&language=" + lang +
		"&pincode=" + pincode +
		"&ward=" + ward +
		"&nagarsevak_id=" + nsID +
		"&nagarsevak_name=" + nsName +
		"&nagarsevak_phone=" + nsPhone

	url := h.playURL("record_start", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.Play(url),
			plivo.RecordWithBeep(action, 120, "#"),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.Speak("After the beep, please describe your complaint and mention the address or nearby landmark. Press hash when done.", lang),
			plivo.RecordWithBeep(action, 120, "#"),
		))
	}
}

func (h *PlivoHandler) corporatorConnect(c *gin.Context, phone, lang, pincode, ward, nsID, nsName, nsPhone string) {
	if nsName == "" {
		nsName = "your corporator"
	}

	if nsPhone == "" {
		h.returnComplaintOffer(c, phone, lang, pincode, ward, nsID, nsName, "")
		return
	}

	dialAction := h.baseURL + "/ivr/plivo/dial-status?phone=" + phone +
		"&language=" + lang +
		"&pincode=" + pincode +
		"&ward=" + ward +
		"&nagarsevak_id=" + nsID +
		"&nagarsevak_name=" + nsName +
		"&nagarsevak_phone=" + nsPhone

	c.String(http.StatusOK, plivo.Response(
		plivo.Speak("Connecting you to "+nsName+". Please hold.", lang),
		plivo.DialWithAction(nsPhone, dialAction),
	))
}

func (h *PlivoHandler) wardInputRetry(c *gin.Context, phone, lang, pincode, retryStr string) {
	retry, _ := strconv.Atoi(retryStr)
	if retry >= h.maxRetries {
		h.returnWhatsAppPrompt(c, lang)
		return
	}

	action := h.baseURL + "/ivr/plivo/ward-input?phone=" + phone + "&language=" + lang + "&pincode=" + pincode + "&retry=" + strconv.Itoa(retry+1)

	url := h.playURL("ward_input", lang)
	if url != "" {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigits(action, 10, 15, plivo.Play(url)),
		))
	} else {
		c.String(http.StatusOK, plivo.Response(
			plivo.GetDigits(action, 10, 15, plivo.Speak("We could not find a matching ward. Please try again. Enter your ward number.", lang)),
		))
	}
}

func (h *PlivoHandler) returnWardMenu(c *gin.Context, phone, lang, pincode string, matches []models.WardMatch) {
	var wards []string
	var ttsParts []string
	for i, m := range matches {
		wards = append(wards, m.Ward)
		ttsParts = append(ttsParts, "Press "+strconv.Itoa(i+1)+" for "+m.Ward)
	}

	action := h.baseURL + "/ivr/plivo/ward-select?phone=" + phone + "&language=" + lang +
		"&pincode=" + pincode + "&wards=" + strings.Join(wards, ",")

	c.String(http.StatusOK, plivo.Response(
		plivo.GetDigits(action, 1, 10, plivo.Speak("Multiple wards found. "+strings.Join(ttsParts, ". ")+".", lang)),
	))
}

func (h *PlivoHandler) returnNagarsevakMenu(c *gin.Context, phone, lang, pincode, ward string, nagarsevaks []models.NagarsevakRecord) {
	var ids []string
	var ttsParts []string
	for i, ns := range nagarsevaks {
		ids = append(ids, ns.ID.String())
		ttsParts = append(ttsParts, "Press "+strconv.Itoa(i+1)+" for "+ns.Name)
	}

	action := h.baseURL + "/ivr/plivo/nagarsevak-select?phone=" + phone + "&language=" + lang +
		"&pincode=" + pincode + "&ward=" + ward + "&ids=" + strings.Join(ids, ",")

	c.String(http.StatusOK, plivo.Response(
		plivo.GetDigits(action, 1, 10, plivo.Speak("Multiple corporators found. "+strings.Join(ttsParts, ". ")+".", lang)),
	))
}

func (h *PlivoHandler) returnWhatsAppPrompt(c *gin.Context, lang string) {
	c.String(http.StatusOK, plivo.Response(
		plivo.Speak("We could not find your information. Please contact us on WhatsApp for assistance.", lang),
		plivo.Hangup(),
	))
}

// --------------- helpers ---------------

func resolveLanguage(digits string) string {
	switch digits {
	case "1":
		return "english"
	case "2":
		return "hindi"
	case "3":
		return "marathi"
	default:
		return "english"
	}
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	return phone
}

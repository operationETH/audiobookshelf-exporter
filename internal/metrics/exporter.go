package metrics

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/operationeth/audiobookshelf-exporter/internal/api"
	"github.com/prometheus/client_golang/prometheus"
)

type Exporter struct {
	client *api.Client

	recentSessionsLimit int
	recentItemsLimit    int

	up          prometheus.Gauge
	users       prometheus.Gauge
	libItems    *prometheus.GaugeVec
	lastSuccess prometheus.Gauge
	lastTime    prometheus.Gauge
	duration    prometheus.Summary

	userListeningSeconds    *prometheus.GaugeVec
	userSessionsTotal       *prometheus.GaugeVec
	libraryListeningSeconds *prometheus.GaugeVec
	librarySessionsTotal    *prometheus.GaugeVec
	bookListeningSeconds    *prometheus.GaugeVec
	deviceListeningSeconds  *prometheus.GaugeVec
	weekdayListeningSeconds *prometheus.GaugeVec
	hourListeningSeconds    *prometheus.GaugeVec
	sessionsTotal           prometheus.Gauge

	openSessionsTotal                  prometheus.Gauge
	openSessionInfo                    *prometheus.GaugeVec
	openSessionCurrentTimeSeconds      *prometheus.GaugeVec
	openSessionDurationSeconds         *prometheus.GaugeVec
	openSessionProgressPercent         *prometheus.GaugeVec
	openSessionTimeListeningSeconds    *prometheus.GaugeVec
	openSessionStartedTimestampSeconds *prometheus.GaugeVec
	openSessionUpdatedTimestampSeconds *prometheus.GaugeVec

	recentSessionInfo                    *prometheus.GaugeVec
	recentSessionListeningSeconds        *prometheus.GaugeVec
	recentSessionProgressPercent         *prometheus.GaugeVec
	recentSessionStartedTimestampSeconds *prometheus.GaugeVec
	recentSessionUpdatedTimestampSeconds *prometheus.GaugeVec

	recentAddedInfo             *prometheus.GaugeVec
	recentAddedTimestampSeconds *prometheus.GaugeVec
	recentAddedDurationSeconds  *prometheus.GaugeVec
	recentAddedSizeBytes        *prometheus.GaugeVec
}

func NewExporter(c *api.Client) *Exporter {
	e := &Exporter{
		client:              c,
		recentSessionsLimit: envInt("ABS_RECENT_SESSIONS_LIMIT", 10),
		recentItemsLimit:    envInt("ABS_RECENT_ITEMS_LIMIT", 10),
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_up",
			Help: "1 if Audiobookshelf was reachable during the last scrape",
		}),
		users: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_users_total",
			Help: "Number of users in Audiobookshelf",
		}),
		libItems: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_library_items_total",
			Help: "Total items per library (if provided by API, otherwise 0)",
		}, []string{"library_id", "library_name"}),
		lastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_last_scrape_success",
			Help: "1 if last scrape was successful",
		}),
		lastTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_last_scrape_timestamp_seconds",
			Help: "Unix timestamp of last scrape",
		}),
		duration: prometheus.NewSummary(prometheus.SummaryOpts{
			Name: "audiobookshelf_scrape_duration_seconds",
			Help: "Duration of Audiobookshelf exporter scrape in seconds",
		}),

		userListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_user_listening_seconds_total",
			Help: "Total listening time per user across all sessions",
		}, []string{"user"}),

		userSessionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_user_sessions_total",
			Help: "Total number of sessions per user",
		}, []string{"user"}),

		libraryListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_library_listening_seconds_total",
			Help: "Total listening time per library",
		}, []string{"library_id", "library_name"}),

		librarySessionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_library_sessions_total",
			Help: "Total number of sessions per library",
		}, []string{"library_id", "library_name"}),

		bookListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_book_listening_seconds_total",
			Help: "Total listening time per media item title",
		}, []string{"media_type", "title"}),

		deviceListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_device_listening_seconds_total",
			Help: "Total listening time per client / device model",
		}, []string{"client", "model"}),

		weekdayListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_weekday_listening_seconds_total",
			Help: "Total listening time grouped by day of week",
		}, []string{"day"}),

		hourListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_hour_listening_seconds_total",
			Help: "Total listening time grouped by session start hour of day",
		}, []string{"hour"}),

		sessionsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_sessions_total",
			Help: "Total number of sessions returned by /api/sessions",
		}),

		openSessionsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_sessions_total",
			Help: "Number of active/open Audiobookshelf playback sessions",
		}),
		openSessionInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_info",
			Help: "Information about active/open Audiobookshelf playback sessions. Value is always 1.",
		}, []string{"session_id", "user", "user_id", "library_id", "library_name", "media_type", "title", "author", "client", "model", "device_name"}),
		openSessionCurrentTimeSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_current_time_seconds",
			Help: "Current playback position for active/open sessions in seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		openSessionDurationSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_duration_seconds",
			Help: "Total media duration for active/open sessions in seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		openSessionProgressPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_progress_percent",
			Help: "Current playback progress for active/open sessions as a percent",
		}, []string{"session_id", "user", "title", "client", "model"}),
		openSessionTimeListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_time_listening_seconds",
			Help: "Time listened during the active/open session in seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		openSessionStartedTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_started_timestamp_seconds",
			Help: "Start timestamp for active/open sessions as Unix seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		openSessionUpdatedTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_open_session_updated_timestamp_seconds",
			Help: "Last update timestamp for active/open sessions as Unix seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),

		recentSessionInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_session_info",
			Help: "Information about recently listened Audiobookshelf sessions. Value is always 1.",
		}, []string{"session_id", "user", "user_id", "library_id", "library_name", "media_type", "title", "author", "date", "day", "client", "model", "device_name"}),
		recentSessionListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_session_listening_seconds",
			Help: "Listening time for recently listened sessions in seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		recentSessionProgressPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_session_progress_percent",
			Help: "Playback progress for recently listened sessions as a percent",
		}, []string{"session_id", "user", "title", "client", "model"}),
		recentSessionStartedTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_session_started_timestamp_seconds",
			Help: "Start timestamp for recently listened sessions as Unix seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),
		recentSessionUpdatedTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_session_updated_timestamp_seconds",
			Help: "Last update timestamp for recently listened sessions as Unix seconds",
		}, []string{"session_id", "user", "title", "client", "model"}),

		recentAddedInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_added_info",
			Help: "Information about recently added Audiobookshelf library items. Value is always 1.",
		}, []string{"item_id", "library_id", "library_name", "media_type", "title", "author", "series", "year"}),
		recentAddedTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_added_timestamp_seconds",
			Help: "Added timestamp for recently added library items as Unix seconds",
		}, []string{"item_id", "library_name", "title", "author", "media_type"}),
		recentAddedDurationSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_added_duration_seconds",
			Help: "Duration for recently added library items in seconds",
		}, []string{"item_id", "library_name", "title", "author", "media_type"}),
		recentAddedSizeBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "audiobookshelf_recent_added_size_bytes",
			Help: "Size for recently added library items in bytes",
		}, []string{"item_id", "library_name", "title", "author", "media_type"}),
	}

	prometheus.MustRegister(
		e.up,
		e.users,
		e.libItems,
		e.lastSuccess,
		e.lastTime,
		e.duration,
		e.userListeningSeconds,
		e.userSessionsTotal,
		e.libraryListeningSeconds,
		e.librarySessionsTotal,
		e.bookListeningSeconds,
		e.deviceListeningSeconds,
		e.weekdayListeningSeconds,
		e.hourListeningSeconds,
		e.sessionsTotal,
		e.openSessionsTotal,
		e.openSessionInfo,
		e.openSessionCurrentTimeSeconds,
		e.openSessionDurationSeconds,
		e.openSessionProgressPercent,
		e.openSessionTimeListeningSeconds,
		e.openSessionStartedTimestampSeconds,
		e.openSessionUpdatedTimestampSeconds,
		e.recentSessionInfo,
		e.recentSessionListeningSeconds,
		e.recentSessionProgressPercent,
		e.recentSessionStartedTimestampSeconds,
		e.recentSessionUpdatedTimestampSeconds,
		e.recentAddedInfo,
		e.recentAddedTimestampSeconds,
		e.recentAddedDurationSeconds,
		e.recentAddedSizeBytes,
	)

	return e
}

func (e *Exporter) Scrape() {
	start := time.Now()
	success := true
	absReachable := false

	users, err := e.client.Users()
	userNames := map[string]string{}
	if err != nil {
		log.Println("users:", err)
		success = false
	} else {
		absReachable = true
		e.users.Set(float64(len(users)))
		for _, u := range users {
			if u.ID != "" && u.Username != "" {
				userNames[u.ID] = u.Username
			}
		}
	}

	libs, err := e.client.Libraries()
	libNames := map[string]string{}

	if err != nil {
		log.Println("libs:", err)
		success = false
	} else {
		absReachable = true
		e.libItems.Reset()

		for _, l := range libs {
			libNames[l.ID] = l.Name

			d, err := e.client.LibraryDetail(l.ID)
			if err != nil {
				log.Println("detail:", err)
				success = false
				continue
			}
			e.libItems.WithLabelValues(l.ID, l.Name).Set(float64(d.TotalItems))
		}
	}

	sessions, err := e.client.Sessions()
	if err != nil {
		log.Println("sessions:", err)
		success = false
	} else {
		absReachable = true
		userListening := make(map[string]float64)
		userSessions := make(map[string]float64)
		libraryListening := make(map[string]float64)
		librarySessions := make(map[string]float64)
		bookListening := make(map[[2]string]float64)
		deviceListening := make(map[[2]string]float64)
		weekdayListening := make(map[string]float64)
		hourListening := make(map[string]float64)

		for _, s := range sessions {
			listened := s.TimeListening
			if listened <= 0 {
				continue
			}

			username := s.UserID
			if s.User != nil && s.User.Username != "" {
				username = s.User.Username
			}
			if username == "" {
				username = "unknown"
			}

			libID := s.LibraryID
			if libID == "" {
				libID = "unknown"
			}
			libName := libNames[libID]
			if libName == "" {
				libName = libID
			}

			mediaType := s.MediaType
			if mediaType == "" {
				mediaType = "unknown"
			}

			title := ""
			if s.MediaMetadata != nil && s.MediaMetadata.Title != "" {
				title = s.MediaMetadata.Title
			} else {
				title = "unknown"
			}

			var client, model string
			if s.DeviceInfo != nil {
				if s.DeviceInfo.ClientName != "" {
					client = s.DeviceInfo.ClientName
				} else {
					client = "unknown"
				}
				if s.DeviceInfo.Model != "" {
					model = s.DeviceInfo.Model
				} else {
					model = "unknown"
				}
			} else {
				client = "unknown"
				model = "unknown"
			}

			day := s.DayOfWeek
			if day == "" {
				day = "unknown"
			}

			userListening[username] += listened
			userSessions[username]++

			libraryKey := libID + "||" + libName
			libraryListening[libraryKey] += listened
			librarySessions[libraryKey]++

			bookKey := [2]string{mediaType, title}
			bookListening[bookKey] += listened

			dmKey := [2]string{client, model}
			deviceListening[dmKey] += listened

			weekdayListening[day] += listened
			hourListening[sessionHour(s)] += listened
		}

		e.userListeningSeconds.Reset()
		e.userSessionsTotal.Reset()
		e.libraryListeningSeconds.Reset()
		e.librarySessionsTotal.Reset()
		e.bookListeningSeconds.Reset()
		e.deviceListeningSeconds.Reset()
		e.weekdayListeningSeconds.Reset()
		e.hourListeningSeconds.Reset()

		for user, secs := range userListening {
			e.userListeningSeconds.WithLabelValues(user).Set(secs)
		}
		for user, count := range userSessions {
			e.userSessionsTotal.WithLabelValues(user).Set(count)
		}

		for key, secs := range libraryListening {
			parts := splitOnce(key, "||")
			libID := parts[0]
			libName := parts[1]
			e.libraryListeningSeconds.WithLabelValues(libID, libName).Set(secs)
		}
		for key, count := range librarySessions {
			parts := splitOnce(key, "||")
			libID := parts[0]
			libName := parts[1]
			e.librarySessionsTotal.WithLabelValues(libID, libName).Set(count)
		}

		for bookKey, secs := range bookListening {
			mediaType := bookKey[0]
			title := bookKey[1]
			e.bookListeningSeconds.WithLabelValues(mediaType, title).Set(secs)
		}

		for dmKey, secs := range deviceListening {
			client := dmKey[0]
			model := dmKey[1]
			e.deviceListeningSeconds.WithLabelValues(client, model).Set(secs)
		}

		for day, secs := range weekdayListening {
			e.weekdayListeningSeconds.WithLabelValues(day).Set(secs)
		}

		for hour := 0; hour < 24; hour++ {
			hourLabel := fmt.Sprintf("%02d:00", hour)
			e.hourListeningSeconds.WithLabelValues(hourLabel).Set(hourListening[hourLabel])
		}

		e.sessionsTotal.Set(float64(len(sessions)))
	}

	openSessions, err := e.client.OpenSessions()
	if err != nil {
		log.Println("open sessions:", err)
		success = false
	} else {
		absReachable = true
		e.openSessionsTotal.Set(float64(len(openSessions)))
		e.openSessionInfo.Reset()
		e.openSessionCurrentTimeSeconds.Reset()
		e.openSessionDurationSeconds.Reset()
		e.openSessionProgressPercent.Reset()
		e.openSessionTimeListeningSeconds.Reset()
		e.openSessionStartedTimestampSeconds.Reset()
		e.openSessionUpdatedTimestampSeconds.Reset()

		for _, s := range openSessions {
			labels := sessionLabels(s, userNames, libNames)
			e.openSessionInfo.WithLabelValues(
				labels.sessionID,
				labels.user,
				labels.userID,
				labels.libraryID,
				labels.libraryName,
				labels.mediaType,
				labels.title,
				labels.author,
				labels.client,
				labels.model,
				labels.deviceName,
			).Set(1)

			e.openSessionCurrentTimeSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(s.CurrentTime)
			e.openSessionDurationSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(s.Duration)
			e.openSessionProgressPercent.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(progressPercent(s.CurrentTime, s.Duration))
			e.openSessionTimeListeningSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(s.TimeListening)
			e.openSessionStartedTimestampSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(msToSeconds(s.StartedAt))
			e.openSessionUpdatedTimestampSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(msToSeconds(s.UpdatedAt))
		}
	}

	recentSessions, err := e.client.RecentSessions(e.recentSessionsLimit)
	if err != nil {
		log.Println("recent sessions:", err)
		success = false
	} else {
		absReachable = true
		e.recentSessionInfo.Reset()
		e.recentSessionListeningSeconds.Reset()
		e.recentSessionProgressPercent.Reset()
		e.recentSessionStartedTimestampSeconds.Reset()
		e.recentSessionUpdatedTimestampSeconds.Reset()

		for _, s := range recentSessions {
			labels := sessionLabels(s, userNames, libNames)
			day := valueOrUnknown(s.DayOfWeek)
			date := valueOrUnknown(s.Date)

			e.recentSessionInfo.WithLabelValues(
				labels.sessionID,
				labels.user,
				labels.userID,
				labels.libraryID,
				labels.libraryName,
				labels.mediaType,
				labels.title,
				labels.author,
				date,
				day,
				labels.client,
				labels.model,
				labels.deviceName,
			).Set(1)

			e.recentSessionListeningSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(s.TimeListening)
			e.recentSessionProgressPercent.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(progressPercent(s.CurrentTime, s.Duration))
			e.recentSessionStartedTimestampSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(msToSeconds(s.StartedAt))
			e.recentSessionUpdatedTimestampSeconds.WithLabelValues(labels.sessionID, labels.user, labels.title, labels.client, labels.model).Set(msToSeconds(s.UpdatedAt))
		}
	}

	e.recentAddedInfo.Reset()
	e.recentAddedTimestampSeconds.Reset()
	e.recentAddedDurationSeconds.Reset()
	e.recentAddedSizeBytes.Reset()

	for _, l := range libs {
		items, err := e.client.RecentLibraryItems(l.ID, e.recentItemsLimit)
		if err != nil {
			log.Println("recent items:", err)
			success = false
			continue
		}
		absReachable = true

		for _, item := range items {
			itemID := valueOrUnknown(item.ID)
			libraryID := valueOrUnknown(item.LibraryID)
			libraryName := valueOrUnknown(libNames[item.LibraryID])
			mediaType := valueOrUnknown(item.MediaType)
			title := valueOrUnknown(item.Media.Metadata.Title)
			author := valueOrUnknown(item.Media.Metadata.AuthorName)
			series := valueOrUnknown(item.Media.Metadata.SeriesName)
			year := valueOrUnknown(item.Media.Metadata.PublishedYear)

			e.recentAddedInfo.WithLabelValues(itemID, libraryID, libraryName, mediaType, title, author, series, year).Set(1)
			e.recentAddedTimestampSeconds.WithLabelValues(itemID, libraryName, title, author, mediaType).Set(msToSeconds(item.AddedAt))
			e.recentAddedDurationSeconds.WithLabelValues(itemID, libraryName, title, author, mediaType).Set(item.Media.Duration)
			e.recentAddedSizeBytes.WithLabelValues(itemID, libraryName, title, author, mediaType).Set(item.Media.Size)
		}
	}

	if absReachable {
		e.up.Set(1)
	} else {
		e.up.Set(0)
	}

	if success {
		e.lastSuccess.Set(1)
	} else {
		e.lastSuccess.Set(0)
	}

	e.lastTime.Set(float64(time.Now().Unix()))
	e.duration.Observe(time.Since(start).Seconds())
}

func (e *Exporter) Run(interval time.Duration) {
	e.Scrape()
	t := time.NewTicker(interval)
	for range t.C {
		e.Scrape()
	}
}

type normalizedSessionLabels struct {
	sessionID   string
	user        string
	userID      string
	libraryID   string
	libraryName string
	mediaType   string
	title       string
	author      string
	client      string
	model       string
	deviceName  string
}

func sessionHour(s api.Session) string {
	hour := int(s.StartTime) / 3600
	if hour >= 0 && hour <= 23 {
		return fmt.Sprintf("%02d:00", hour)
	}

	if s.StartedAt > 0 {
		return time.UnixMilli(s.StartedAt).Local().Format("15:00")
	}

	return "unknown"
}

func sessionLabels(s api.Session, userNames map[string]string, libNames map[string]string) normalizedSessionLabels {
	userID := valueOrUnknown(s.UserID)
	user := userNames[s.UserID]
	if user == "" && s.User != nil {
		user = s.User.Username
	}
	if user == "" {
		user = userID
	}
	user = valueOrUnknown(user)

	libraryID := valueOrUnknown(s.LibraryID)
	libraryName := libNames[s.LibraryID]
	if libraryName == "" {
		libraryName = libraryID
	}
	libraryName = valueOrUnknown(libraryName)

	title := s.DisplayTitle
	if title == "" && s.MediaMetadata != nil {
		title = s.MediaMetadata.Title
	}
	title = valueOrUnknown(title)

	author := valueOrUnknown(s.DisplayAuthor)
	mediaType := valueOrUnknown(s.MediaType)

	client := "unknown"
	model := "unknown"
	deviceName := "unknown"
	if s.DeviceInfo != nil {
		client = valueOrUnknown(s.DeviceInfo.ClientName)
		model = valueOrUnknown(s.DeviceInfo.Model)
		deviceName = valueOrUnknown(s.DeviceInfo.DeviceName)
		if model == "unknown" && deviceName != "unknown" {
			model = deviceName
		}
	}

	return normalizedSessionLabels{
		sessionID:   valueOrUnknown(s.ID),
		user:        user,
		userID:      userID,
		libraryID:   libraryID,
		libraryName: libraryName,
		mediaType:   mediaType,
		title:       title,
		author:      author,
		client:      client,
		model:       model,
		deviceName:  deviceName,
	}
}

func valueOrUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return v
}

func progressPercent(current, duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	return (current / duration) * 100
}

func msToSeconds(ms int64) float64 {
	if ms <= 0 {
		return 0
	}
	return float64(ms) / 1000
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitOnce(s, sep string) [2]string {
	idx := -1
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return [2]string{s, ""}
	}
	return [2]string{s[:idx], s[idx+len(sep):]}
}

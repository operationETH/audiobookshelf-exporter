package metrics

import (
    "log"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/operationETH/audiobookshelf-exporter/internal/api"
)

type Exporter struct {
    client *api.Client

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
    sessionsTotal           prometheus.Gauge
}

func NewExporter(c *api.Client) *Exporter {
    e := &Exporter{
        client: c,
        up: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "audiobookshelf_up",
            Help: "1 if exporter successfully scraped Audiobookshelf",
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
            Help: "Total listening time per book title",
        }, []string{"title"}),

        deviceListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
            Name: "audiobookshelf_device_listening_seconds_total",
            Help: "Total listening time per client / device model",
        }, []string{"client", "model"}),

        weekdayListeningSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
            Name: "audiobookshelf_weekday_listening_seconds_total",
            Help: "Total listening time grouped by day of week",
        }, []string{"day"}),

        sessionsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "audiobookshelf_sessions_total",
            Help: "Total number of sessions returned by /api/sessions",
        }),
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
        e.sessionsTotal,
    )

    return e
}

func (e *Exporter) Scrape() {
    start := time.Now()
    success := true

    e.up.Set(0)
    e.lastSuccess.Set(0)


    users, err := e.client.Users()
    if err != nil {
        log.Println("users:", err)
        success = false
    } else {
        e.users.Set(float64(len(users)))
    }


    libs, err := e.client.Libraries()
    libNames := map[string]string{}

    if err != nil {
        log.Println("libs:", err)
        success = false
    } else {
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
        userListening := make(map[string]float64)
        userSessions := make(map[string]float64)
        libraryListening := make(map[string]float64)
        librarySessions := make(map[string]float64)
        bookListening := make(map[string]float64)
        deviceListening := make(map[[2]string]float64)
        weekdayListening := make(map[string]float64)

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

            bookListening[title] += listened

            dmKey := [2]string{client, model}
            deviceListening[dmKey] += listened

            weekdayListening[day] += listened
        }

        e.userListeningSeconds.Reset()
        e.userSessionsTotal.Reset()
        e.libraryListeningSeconds.Reset()
        e.librarySessionsTotal.Reset()
        e.bookListeningSeconds.Reset()
        e.deviceListeningSeconds.Reset()
        e.weekdayListeningSeconds.Reset()


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

        for title, secs := range bookListening {
            e.bookListeningSeconds.WithLabelValues(title).Set(secs)
        }

        for dmKey, secs := range deviceListening {
            client := dmKey[0]
            model := dmKey[1]
            e.deviceListeningSeconds.WithLabelValues(client, model).Set(secs)
        }

        for day, secs := range weekdayListening {
            e.weekdayListeningSeconds.WithLabelValues(day).Set(secs)
        }

        e.sessionsTotal.Set(float64(len(sessions)))
    }

    if success {
        e.up.Set(1)
        e.lastSuccess.Set(1)
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

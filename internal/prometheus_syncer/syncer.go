package prometheus_syncer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"grafana-dashboards-downloader/internal/config"
	"grafana-dashboards-downloader/internal/http_downloader"
	"grafana-dashboards-downloader/internal/trash_collector"
	"log/slog"
	"os"
	"path/filepath"
)

func NewPrometheusSyncer(logger *slog.Logger, config config.PrometheusConfig) *PrometheusSyncer {
	return &PrometheusSyncer{
		logger:               logger,
		config:               config,
		downloadedFilesCache: make(map[string]string),
	}
}

type PrometheusSyncer struct {
	logger               *slog.Logger
	config               config.PrometheusConfig
	downloadedFilesCache map[string]string
}

func (p *PrometheusSyncer) Sync() {
	trashCollector := trash_collector.NewTrashCollector(p.config.PrometheusRulesPath)

	for _, prometheusRule := range p.config.PrometheusRules {
		content, err := http_downloader.Download(prometheusRule.HTTPSource.Url)
		if err != nil {
			p.logger.Error("couldn't download prometheus rules", "error", err, "url", prometheusRule.HTTPSource.Url)
			continue
		}

		filename := p.generateFilename(prometheusRule)

		cachedValue, exists := p.downloadedFilesCache[filename]
		fmt.Println(4, p.downloadedFilesCache)
		if !exists || cachedValue != string(content) {
			fullPath := filepath.Join(p.config.PrometheusRulesPath, filename)
			err = os.WriteFile(fullPath, content, 0644)
			if err != nil {
				p.logger.Error(
					"couldn't save prometheus rule file to disk",
					"error", err,
					"url", prometheusRule.HTTPSource.Url,
					"path", fullPath,
				)
				continue
			}

			p.downloadedFilesCache[filename] = string(content)
			fmt.Println(5, p.downloadedFilesCache)
		}

		trashCollector.AddKnownFile(filename)
	}

	err := trashCollector.PickUpTrash()
	if err != nil {
		p.logger.Error("couldn't delete unknown files", "error", err)
	}
}

func (p *PrometheusSyncer) generateFilename(prometheusRule config.PrometheusRuleConfig) string {
	md5sum := md5.New()
	md5sum.Write([]byte(prometheusRule.HTTPSource.Url))
	filenameBase := hex.EncodeToString(md5sum.Sum(nil))
	return fmt.Sprintf("%s.yml", filenameBase)
}

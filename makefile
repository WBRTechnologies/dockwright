CONFIG_FILE ?= dockwright.conf
-include $(CONFIG_FILE)

# Export so shell recipes also see CHARTS_DIR
export CHARTS_DIR

CHART_SRC_ROOT ?= helm-base-charts

.PHONY: all package-charts install uninstall clean

all: install

package-charts:
	@set -euo pipefail; \
	if [ -z "$(CHARTS_DIR)" ]; then \
	  echo "CHARTS_DIR not set in $(CONFIG_FILE)" >&2; exit 1; \
	fi; \
	if [ ! -d "$(CHART_SRC_ROOT)" ]; then \
	  echo "Chart source directory '$(CHART_SRC_ROOT)' not found" >&2; exit 1; \
	fi; \
	mkdir -p "$(CHARTS_DIR)"; \
	found=0; \
	for d in "$(CHART_SRC_ROOT)"/*; do \
	  [ -d "$$d" ] || continue; \
	  found=1; \
	  name=$$(basename "$$d"); \
	  echo "Packaging $$name"; \
	  helm package "$$d" -d "$(CHARTS_DIR)"; \
	  latest=$$(ls -1 "$(CHARTS_DIR)/$$name"-*.tgz 2>/dev/null | sort | tail -n1); \
	  [ -n "$$latest" ] || { echo "No package produced for $$name" >&2; exit 1; }; \
	  cp "$$latest" "$(CHARTS_DIR)/$$name.tgz"; \
	done; \
	[ "$$found" -eq 1 ] || { echo "No charts found under $(CHART_SRC_ROOT)" >&2; exit 1; }; \
	echo "Charts packaged into $(CHARTS_DIR)"

install: package-charts
	@set -euo pipefail; \
	mkdir -p /usr/local/bin /etc; \
	install -m 0755 deployer.sh /usr/local/bin/dockwright; \
	install -m 0644 "$(CONFIG_FILE)" /etc/dockwright.conf; \
	echo "Installed deployer and config"

uninstall:
	@set -euo pipefail; \
	rm -f /usr/local/bin/dockwright; \
	rm -f /etc/dockwright.conf; \
	[ -n "$(CHARTS_DIR)" ] && [ -d "$(CHARTS_DIR)" ] && rm -rf "$(CHARTS_DIR)"; \
	echo "Uninstalled dockwright"

clean:
	@set -euo pipefail; \
	[ -n "$(CHARTS_DIR)" ] && [ -d "$(CHARTS_DIR)" ] && rm -f "$(CHARTS_DIR)"/*.tgz || true; \
	echo "Clean complete"
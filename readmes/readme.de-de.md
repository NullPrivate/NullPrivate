# NullPrivate

NullPrivate ist eine Abspaltung von _AdGuardHome_, die entwickelt wurde, um eine SaaS-gehostete Version mit erweiterten Funktionen und Anpassungsmöglichkeiten zu bieten. Es wird gehostet auf [AdGuard Private](https://nullprivate.com).

## Wichtige Funktionen

### Ursprüngliche Funktionen

1. **Netzwerkweite Werbeblockierung**

   - Blockiert Werbung und Tracker auf allen Geräten in Ihrem Netzwerk.
   - Funktioniert als DNS-Server, der Tracking-Domains zu einem „Black Hole“ umleitet.

2. **Benutzerdefinierte Filterregeln**

   - Fügen Sie Ihre eigenen benutzerdefinierten Filterregeln hinzu.
   - Überwachen und steuern Sie die Netzwerkaktivität.

3. **Unterstützung für verschlüsseltes DNS**

   - Unterstützt DNS-über-HTTPS, DNS-über-TLS und DNSCrypt.

4. **Integrierter DHCP-Server**

   - Bietet DHCP-Server-Funktionalität aus dem Stand.

5. **Konfiguration pro Kunde**

   - Konfigurieren Sie Einstellungen für einzelne Geräte.

6. **Elternkontrolle**

   - Blockiert Erwachsenen-Domains und erzwingt Safe Search bei Suchmaschinen.

7. **Plattformübergreifende Kompatibilität**

   - Läuft auf Linux, macOS, Windows und mehr.

8. **Datenschutzorientiert**
   - Sammelt keine Nutzungsstatistiken oder sendet Daten, es sei denn, es ist explizit konfiguriert.

### Neue Funktionen von NullPrivate

1. **DNS-Routing mit Regelisten**

   - Passen Sie das DNS-Routing mit in der Konfigurationsdatei definierten Regelisten an.
   - Unterstützt Drittanbieter-Regeln wie [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat).

2. **Anwendungspezifische Blockierungsregellisten**

   - Konfigurieren Sie das Blockieren von Quellen aus spezifischen Anwendungen.
   - Unterstützt Drittanbieter-Konfigurationen für flexible Verwaltung.

3. **Dynamisches DNS (DDNS)**

   - Bietet dynamische Domänenauflösungsfähigkeiten für verschiedene Szenarien.

4. **Erweitertes Ratenlimit**

   - Implementiert effizientes Verkehrsmanagement und Kontrollmaßnahmen.

5. **Erweiterte Bereitstellungsfunktionen**
   - Unterstützung für Lastverteilung.
   - Automatische Zertifikatspflege.
   - Optimierte Netzwerkverbindungen.

Für detaillierte Dokumentation besuchen Sie: [NullPrivate Dokumentation](https://nullprivate.com/docs/)

## Nutzung

### Binärdatei herunterladen

Sie können die Binärdatei direkt von der [Releases](https://github.com/NullPrivate/NullPrivate/releases)-Seite herunterladen. Nach dem Herunterladen befolgen Sie diese Schritte, um sie auszuführen:

```bash
./NullPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Docker-Image verwenden

Alternativ können Sie das Docker-Image verwenden, das auf [Docker Hub](https://hub.docker.com/repository/docker/nullprivate/nullprivate) verfügbar ist:

```bash
docker run --rm --name NullPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf nullprivate/nullprivate:latest
```
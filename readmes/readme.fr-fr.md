# NullPrivate

NullPrivate est une version dérivée de _AdGuardHome_, conçue pour fournir une version hébergée en SaaS avec des fonctionnalités améliorées et une personnalisation accrue. Elle est hébergée sur [AdGuard Private](https://nullprivate.com).

## Fonctionnalités principales

### Fonctionnalités d'origine

1. **Blocage des publicités sur l'ensemble du réseau**

   - Bloque les publicités et les trackers sur tous les appareils de votre réseau.
   - Fonctionne comme un serveur DNS qui redirige les domaines de suivi vers un « trou noir ».

2. **Règles de filtrage personnalisées**

   - Ajoutez vos propres règles de filtrage personnalisées.
   - Surveillez et contrôlez l'activité du réseau.

3. **Support DNS chiffré**

   - Supporte DNS-over-HTTPS, DNS-over-TLS et DNSCrypt.

4. **Serveur DHCP intégré**

   - Fournit une fonctionnalité de serveur DHCP prête à l'emploi.

5. **Configuration par client**

   - Configurez les paramètres pour des appareils individuels.

6. **Contrôle parental**

   - Bloque les domaines pour adultes et impose la recherche sécurisée sur les moteurs de recherche.

7. **Compatibilité multi-plateforme**

   - Fonctionne sur Linux, macOS, Windows, et plus.

8. **Centré sur la vie privée**
   - Ne collecte pas de statistiques d'utilisation ou n'envoie pas de données sauf si explicitement configuré.

### Nouvelles fonctionnalités par NullPrivate

1. **Routage DNS avec listes de règles**

   - Personnalisez le routage DNS en utilisant des listes de règles définies dans le fichier de configuration.
   - Supporte des règles tierces comme [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat).

2. **Listes de règles de blocage spécifiques aux applications**

   - Configurez le blocage des sources provenant d'applications spécifiques.
   - Supporte des configurations tierces pour une gestion flexible.

3. **DNS dynamique (DDNS)**

   - Fournit des capacités de résolution de nom de domaine dynamique pour divers scénarios.

4. **Limitation de débit avancée**

   - Implémente des mesures efficaces de gestion et de contrôle du trafic.

5. **Fonctionnalités de déploiement améliorées**
   - Support de l'équilibrage de charge.
   - Maintenance automatique des certificats.
   - Connexions réseau optimisées.

Pour une documentation détaillée, visitez : [Documentation NullPrivate](https://nullprivate.com/docs/)

## Comment utiliser

### Télécharger le binaire

Vous pouvez télécharger le binaire directement depuis la page [Releases](https://github.com/NullPrivate/NullPrivate/releases). Une fois téléchargé, suivez ces étapes pour l'exécuter :

```bash
./NullPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Utiliser l'image Docker

Alternativement, vous pouvez utiliser l'image Docker disponible sur [Docker Hub](https://hub.docker.com/repository/docker/nullprivate/nullprivate) :

```bash
docker run --rm --name NullPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf nullprivate/nullprivate:latest
```
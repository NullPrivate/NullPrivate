# AdGuardPrivate

AdGuardPrivate es una bifurcación de _AdGuardHome_, diseñada para proporcionar una versión alojada en SaaS con características mejoradas y personalización. Se aloja en [AdGuard Private](https://adguardprivate.com).

## Características clave

### Características originales

1. **Bloqueo de anuncios en toda la red**

   - Bloquea anuncios y rastreadores en todos los dispositivos de tu red.
   - Funciona como un servidor DNS que redirige los dominios de seguimiento a un "agujero negro".

2. **Reglas de filtrado personalizadas**

   - Añade tus propias reglas de filtrado personalizadas.
   - Monitorea y controla la actividad de la red.

3. **Soporte para DNS cifrado**

   - Soporta DNS-over-HTTPS, DNS-over-TLS y DNSCrypt.

4. **Servidor DHCP integrado**

   - Proporciona funcionalidad de servidor DHCP de serie.

5. **Configuración por cliente**

   - Configura ajustes para dispositivos individuales.

6. **Control parental**

   - Bloquea dominios para adultos y aplica búsqueda segura en motores de búsqueda.

7. **Compatibilidad multiplataforma**

   - Funciona en Linux, macOS, Windows y más.

8. **Enfoque en la privacidad**
   - No recopila estadísticas de uso ni envía datos a menos que se configure explícitamente.

### Nuevas características de AdGuardPrivate

1. **Enrutamiento DNS con listas de reglas**

   - Personaliza el enrutamiento DNS utilizando listas de reglas definidas en el archivo de configuración.
   - Soporta reglas de terceros como [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat).

2. **Listas de reglas de bloqueo específicas de aplicaciones**

   - Configura el bloqueo de fuentes de aplicaciones específicas.
   - Soporta configuraciones de terceros para una gestión flexible.

3. **DNS dinámico (DDNS)**

   - Proporciona capacidades de resolución de nombres de dominio dinámicos para varios escenarios.

4. **Limitación de tasa avanzada**

   - Implementa medidas eficientes de gestión y control del tráfico.

5. **Características de despliegue mejoradas**
   - Soporte para balanceo de carga.
   - Mantenimiento automático de certificados.
   - Conexiones de red optimizadas.

Para la documentación detallada, visita: [Documentación de AdGuardPrivate](https://adguardprivate.com/docs/)

## Cómo usar

### Descargar el binario

Puedes descargar el binario directamente desde la página de [Releases](https://github.com/AdGuardPrivate/AdGuardPrivate/releases). Una vez descargado, sigue estos pasos para ejecutarlo:

```bash
./AdGuardPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Usar la imagen de Docker

Alternativamente, puedes usar la imagen de Docker disponible en [Docker Hub](https://hub.docker.com/repository/docker/adguardprivate/adguardprivate):

```bash
docker run --rm --name AdGuardPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf adguardprivate/adguardprivate:latest
```
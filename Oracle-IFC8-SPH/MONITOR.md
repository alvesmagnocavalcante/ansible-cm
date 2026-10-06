# Monitor IFC8

O monitor agora usa Go e instalacao Ansible. Consulte [README.md](README.md) para compilar, configurar `.env`, selecionar o topico e instalar pelo Semaphore.

O template existente `Oracle-IFC8-SPH/playbook.yml` instala o agente Go. Pare qualquer monitor PowerShell antigo antes da migracao para evitar alertas duplicados.

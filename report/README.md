# Heartbeat de hosts Windows com Ansible

Coleta por WinRM informações de CPU, memória, discos, uptime e serviços automáticos
parados. O controlador gera um relatório HTML consolidado, mantém uma cópia local e
envia o conteúdo por e-mail. Hosts inacessíveis aparecem como offline.

## Requisitos

- Controlador Linux ou WSL com Python 3 e Ansible (Ansible não controla hosts a partir do Windows nativo).
- WinRM habilitado nos servidores Windows.
- Acesso SMTP a partir do controlador.

Instale as dependências no controlador:

```bash
python3 -m pip install -r requirements.txt
ansible-galaxy collection install -r requirements.yml
```

## Configuração

1. Edite `inventory/hosts.yml` com os nomes e IPs dos servidores.
2. Exporte as credenciais sem gravá-las no repositório:

```bash
export ANSIBLE_WINDOWS_USER='DOMINIO\\usuario_ansible'
export ANSIBLE_WINDOWS_PASSWORD='senha-windows'
export HEARTBEAT_SMTP_HOST='smtp.exemplo.com'
export HEARTBEAT_SMTP_PORT='587'
export HEARTBEAT_SMTP_USER='monitoramento@exemplo.com'
export HEARTBEAT_SMTP_PASSWORD='senha-smtp'
export HEARTBEAT_EMAIL_FROM='monitoramento@exemplo.com'
export HEARTBEAT_EMAIL_TO='infra@exemplo.com,gestor@exemplo.com'
```

Em produção, prefira Ansible Vault ou um cofre de segredos para essas credenciais.
Se o SMTP não exigir autenticação, deixe usuário e senha vazios. Ajuste os limites
de alerta e `smtp_secure` em `group_vars/all.yml`.

Teste primeiro sem e-mail:

```bash
ansible-playbook playbooks/windows_heartbeat.yml -e send_email=false
```

Execute com envio:

```bash
ansible-playbook playbooks/windows_heartbeat.yml
```

Os relatórios também são salvos em `playbooks/reports/`.

## Agendamento

Após validar a execução manual, agende no `cron` do controlador. Exemplo para rodar
a cada 15 minutos, usando um script protegido que exporte as variáveis:

```cron
*/15 * * * * cd /opt/windows-heartbeat && /opt/windows-heartbeat/run-heartbeat.sh >> /var/log/windows-heartbeat.log 2>&1
```

Não coloque senhas diretamente no `crontab`. Restrinja o script de ambiente com
permissão `0600`, ou use Ansible Vault.

## Observações sobre WinRM

O inventário usa HTTPS/5986 com NTLM e ignora a validação do certificado para
facilitar o primeiro teste. Em produção, instale um certificado confiável e altere
`ansible_winrm_server_cert_validation` para `validate`. Em domínio Kerberos pode ser
uma escolha melhor que NTLM.

# Monitor IFC8 Go + Ansible

Baseado em [monitor-telegram-cm](https://github.com/alvesmagnocavalcante/monitor-telegram-cm): agente Go local, configuracao `.env`, Telegram por hotel e instalacao Ansible no Windows. Este agente verifica a instancia SPH do IFC8 por processo e log, em vez de CPU/RAM/disco.

## Compilar e executar

Use Go 1.27.1 ou superior, no Windows x64:

```powershell
Set-Location C:\ansible-cm\Oracle-IFC8-SPH
go mod download
go test ./...
go vet ./...
go build -trimpath -o bin/monitor-ifc8.exe .
Copy-Item .env.example bin/.env
notepad bin/.env
```

O executavel compilado tambem e incluido no repositorio para a instalacao simples pelo Semaphore. A maquina de destino nao precisa de Go nem de scripts PowerShell em execucao continua. Ao mudar o codigo, recompile e publique o novo executavel junto com o fonte.

Preencha o `.env` com as credenciais e IDs reais. Exemplo usando os IDs do grupo atual do projeto de referencia:

```dotenv
TELEGRAM_BOT_TOKEN=SEU_TOKEN
TELEGRAM_CHAT_ID=-1004428450910
TELEGRAM_TOPIC=Taiba
TELEGRAM_TOPIC_TAIBA_ID=26
MONITOR_INTERVAL=30s
ALERT_REPEAT_INTERVAL=5m
IFC_CONFIG_PATH=C:\Fidelio\Ifc8.Net\IfcApplication\SPH\Ifc8NetConfigSPH.Xml
IFC_LOG_PATH=C:\Fidelio\Ifc8.Net\IfcApplication\SPH\M87POS_SPH_Log.evt
IFC_LOG_MAX_AGE=0s
```

IDs do grupo de referencia: Taiba 26, Charme 27, Magna 29, Acaraizinho 30 e Wind 28. Para outro grupo, configure seus IDs. Os nomes nao diferenciam maiusculas de minusculas. Configure somente o ID do topico selecionado ou todos os IDs num modelo comum. O guia de referencia tambem aceita `TELEGRAM_TOPIC_ID` diretamente; este agente aceita esse formato. Se ambos forem definidos, precisam apontar para o mesmo ID. Nome desconhecido ou ID invalido encerra a inicializacao, evitando envio ao Geral por engano. Topico vazio e sem ID direto envia ao chat.

Para descobrir IDs, envie um comando ao bot dentro de cada topico e consulte `getUpdates`; `chat.id` identifica o grupo e `message_thread_id`, o topico. Use bot sem webhook e sem outro consumidor de atualizacoes. Nao copie tokens de exemplos externos.

O agente procura `.env` ao lado do executavel e, se nao existir, no diretorio atual. `-env` seleciona outro caminho explicitamente. Variaveis existentes no ambiente tem prioridade. As antigas `IFC8_TELEGRAM_TOKEN` e `IFC8_TELEGRAM_CHAT_ID` continuam aceitas como fallback. `.env` e ignorado pelo Git; proteja seu acesso no Windows.

```powershell
# Simulacao: verifica o IFC8, sem enviar e sem exigir token/chat.
.\bin\monitor-ifc8.exe -once -dry-run

# Teste de entrega ao chat/topico configurado.
.\bin\monitor-ifc8.exe -test-telegram

# Monitor continuo; encerre com Ctrl+C.
.\bin\monitor-ifc8.exe

# Configuracao explicita, inclusive fora da pasta de instalacao.
.\bin\monitor-ifc8.exe -env C:\MonitorIFC8\.env
```

`-once` processa os alertas de uma unica leitura; falha de envio retorna codigo de erro. `-dry-run` simula mensagens. Teste de Telegram envia uma mensagem real e encerra. Alteracoes no `.env` exigem reiniciar o agente.

## Estados e alertas

- ONLINE: processo com o argumento exato do arquivo de configuracao e ultimos estados IFC/PMS Alive/Sync.
- OFFLINE: processo ausente ou link/comunicacao diferente de Alive/Sync.
- INDETERMINADO: erro de consulta, log ausente/ilegivel, estados ausentes ou prazo de atualizacao excedido.

Envia OFFLINE/INDETERMINADO, repete a cada 5 minutos e informa recuperacao ONLINE apos uma falha notificada. `ALERT_REPEAT_INTERVAL=0s` desabilita lembretes, como o comportamento por transicoes do agente de referencia. ONLINE inicial nao envia mensagem. O estado so muda apos envio confirmado; falhas tentam novamente na proxima coleta, respeitando `retry_after` do Telegram. O estado fica em memoria: reiniciar pode repetir um alerta. Entrega confirmada pela API nao garante que o usuario leu a mensagem.

O log `.evt` deste IFC8 contem fragmentos XML. O agente le apenas eventos MonItem completos e considera o ultimo estado de cada campo. Arquivos maiores que 16 MiB geram INDETERMINADO. A idade do log nao e usada por padrao porque um log de transicoes pode ficar sem escrita quando a integracao esta funcionando. `IFC_LOG_MAX_AGE=180s` habilita a checagem, classificando atraso como INDETERMINADO. O ultimo estado registrado nao comprova uma conexao ativa neste instante.

O monitor nao inicia nem reinicializa o IFC8. Falha do proprio servidor, parada do monitor ou queda de internet nao pode ser alertada por ele mesmo.

## Instalar pelo Semaphore

O Ansible instala/atualiza; o Go verifica continuamente no Windows. Preserve inventario, WinRM e credenciais ja usados no Semaphore.

1. Playbook: `Oracle-IFC8-SPH/playbook.yml` (ou `Oracle-IFC8-SPH/ansible/install.yml`). Grupo do inventario: `windows_hosts`.
2. Nos Secrets do grupo de variaveis, configure a variavel de ambiente `TELEGRAM_BOT_TOKEN`. Nao coloque o token nas Extra Variables ou no Git.
3. Nas Extra Variables, selecione o hotel. Exemplo:

```json
{
  "agent_topic": "Taiba",
  "agent_interval": "30s",
  "agent_repeat_interval": "5m"
}
```

4. Execute primeiro com Limit para uma maquina IFC8. O inventario deve conter os servidores IFC8, nao todos os PDVs do hotel.

Para outros locais, use `Charme`, `Magna`, `Acaraizinho` ou `Wind`. Se o grupo for diferente, informe `telegram_chat_id` e o mapa `telegram_topics` completo. Ajuste `ifc_config_path` e `ifc_log_path` para instancias diferentes de SPH.

O playbook exige Windows x64 e acesso administrativo. Cria `C:\MonitorIFC8` com acesso somente a Administradores e SYSTEM, copia o exe, grava `.env` com `no_log` e configura a tarefa `MonitorIFC8Telegram` no boot como SYSTEM. Essa conta permite consultar a linha de comando do IFC8 em outra sessao. Nao precisa de usuario conectado nem senha de usuario na tarefa. A API Telegram exige DNS e HTTPS/443.

O runner precisa de `ansible.windows` e `community.windows` previamente instaladas. O `requirements.yml` do repositorio permanece sem downloads automaticos, conforme o problema de DNS observado no Semaphore. Nenhuma colecao e instalada neste fluxo.

Cada execucao reinicia somente a tarefa do monitor. Se falhar depois de parar o monitor, corrija e execute novamente; nao ha rollback automatico. Encerre o antigo script PowerShell e instancias manuais para evitar alertas duplicados. A pasta de instalacao deve ser exclusiva do monitor. O playbook confirma que o agente permanece em execucao, mas simulacao/coleta nao comprova entrega de mensagens; valide com `-test-telegram` no primeiro destino.

Diagnostico no servidor:

```powershell
Get-ScheduledTask -TaskName MonitorIFC8Telegram
Get-ScheduledTaskInfo -TaskName MonitorIFC8Telegram
C:\MonitorIFC8\monitor-ifc8.exe -env C:\MonitorIFC8\.env -once -dry-run
C:\MonitorIFC8\monitor-ifc8.exe -env C:\MonitorIFC8\.env -test-telegram
```

## Estrutura e testes

`main.go` e o ponto de entrada. `internal/agent` contem configuracao, consulta Windows, leitura de log, estado dos alertas e cliente HTTP Telegram. `ansible/install.yml` instala o agente. `bin/monitor-ifc8.exe` e o artefato Windows x64. `go.mod/go.sum` fixam dependencias.

```powershell
go test ./...
go vet ./...
go build -trimpath -o bin/monitor-ifc8.exe .
```

Os testes usam logs temporarios e servidor HTTP local, sem Telegram real. Cobrem queda, recuperacao, leitura incompleta, validacao de topicos, prioridade do ambiente, sigilo do token, erro HTTP, redirecionamentos e limite de envios. A instalacao WinRM e entrega real precisam ser verificadas no seu ambiente.

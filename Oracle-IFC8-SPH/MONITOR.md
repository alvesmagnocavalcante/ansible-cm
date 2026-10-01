# Monitor local do IFC8 SPH

Copie `monitor-ifc8.ps1` para o computador do IFC8 e execute na sessao Windows do usuario `interfaces`:

```powershell
powershell.exe -NoProfile -File .\monitor-ifc8.ps1
```

Verifica a cada 30 segundos o processo da instancia SPH e os ultimos estados `StateLink` e `StateComm` de IFC/PMS no arquivo `M87POS_SPH_Log.evt`.

- `ONLINE`: processo presente e ultimos estados Alive/Sync.
- `OFFLINE`: processo ausente ou estado de link/comunicacao diferente de Alive/Sync.
- `INDETERMINADO`: erro de leitura ou estados ausentes. Tambem gera alerta para sinalizar que o monitor nao conseguiu verificar.

Mostra notificacao do Windows e aviso sonoro ao detectar um problema e repete a cada 5 minutos enquanto persistir. Mostra recuperacao no console. Nao inicia, encerra ou reinicializa o IFC8.

O log informa o ultimo estado registrado, nao garante uma conexao ativa neste instante. A idade do log nao e usada por padrao porque um log de transicoes pode ficar sem escrita mesmo com a integracao funcionando. Para exigir atualizacao em ate 180 segundos, adicione `-MaxLogAgeSeconds 180`; atraso sera reportado como INDETERMINADO.

Para verificar uma unica vez, sem notificacao:

```powershell
.\monitor-ifc8.ps1 -Once
```

Para iniciar automaticamente, crie uma tarefa no Agendador de Tarefas, com gatilho **Ao fazer logon** do usuario `interfaces` e **Executar somente quando o usuario estiver conectado**. Programa: `powershell.exe`; argumentos: `-NoProfile -File "C:\caminho\monitor-ifc8.ps1"`. Use o caminho onde copiou o script. Notificacoes precisam estar habilitadas no Windows; o monitor precisa permanecer em execucao. Sem sessao interativa, use `-NoPopup` para emitir somente no console.

## Telegram

1. Crie um bot no [BotFather](https://t.me/BotFather) usando `/newbot` e guarde o token no servidor. Nao envie o token por chat nem grave no Git.
2. Abra uma conversa com o bot e envie `/start`. Para um grupo, adicione o bot e envie um comando ao bot no grupo.
3. No PowerShell do servidor, informe o token sem exibi-lo:

```powershell
$secret = Read-Host 'Token do bot' -AsSecureString
$env:IFC8_TELEGRAM_TOKEN = [System.Net.NetworkCredential]::new('', $secret).Password
$updates = Invoke-RestMethod -Uri "https://api.telegram.org/bot$($env:IFC8_TELEGRAM_TOKEN)/getUpdates"
$updates.result | ForEach-Object { $_.message.chat } | Select-Object id, title, username -Unique
$env:IFC8_TELEGRAM_CHAT_ID = Read-Host 'ID do chat de destino'
```

`getUpdates` pode nao retornar mensagens quando outro aplicativo consome as atualizacoes ou o bot possui webhook. Use um bot dedicado para este monitor. O ID de grupo geralmente e negativo; copie o valor completo.

4. Teste e execute:

```powershell
.\monitor-ifc8.ps1 -TestTelegram
.\monitor-ifc8.ps1 -Telegram -NoPopup
```

Envia falhas OFFLINE/INDETERMINADO, lembretes a cada 5 minutos e recuperacao ONLINE apos uma falha notificada. Falhas de envio nao interrompem o monitor; tenta novamente apos pelo menos 30 segundos. Nao envia ONLINE na primeira verificacao. `-Once` continua sendo somente uma consulta, sem envio.

As variaveis `$env:` acima duram apenas na sessao atual. Para persistir no perfil do usuario que executara a tarefa:

```powershell
[Environment]::SetEnvironmentVariable('IFC8_TELEGRAM_TOKEN', $env:IFC8_TELEGRAM_TOKEN, 'User')
[Environment]::SetEnvironmentVariable('IFC8_TELEGRAM_CHAT_ID', $env:IFC8_TELEGRAM_CHAT_ID, 'User')
```

Essas variaveis nao sao um cofre criptografado; quem tiver acesso ao perfil pode ler o token. Inicie uma nova sessao do usuario para carregar as variaveis persistidas. Na tarefa agendada, acrescente `-Telegram -NoPopup` aos argumentos; neste modo as notificacoes Telegram nao exigem usuario conectado. Configure a execucao sob a conta onde as variaveis foram salvas e mantenha o monitor em execucao. O servidor precisa resolver `api.telegram.org` e acessar HTTPS/443. Se o servidor ou sua internet cair, o monitor local nao conseguira enviar o alerta.

Referencia: [Telegram Bot API](https://core.telegram.org/bots/api#sendmessage).

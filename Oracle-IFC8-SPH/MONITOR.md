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

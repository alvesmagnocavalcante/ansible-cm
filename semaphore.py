import logging
import time


def configurar_logging() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s | %(levelname)s | %(name)s | %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )


def executar_automacao() -> None:
    logger = logging.getLogger(__name__)

    logger.info("Iniciando automação")

    for etapa in range(1, 6):
        logger.info("Executando etapa %s de 5", etapa)
        time.sleep(1)

    logger.warning("Exemplo de aviso")

    try:
        resultado = 10 / 2
        logger.info("Resultado da operação: %s", resultado)

    except Exception:
        logger.exception("Erro durante a execução")

    logger.info("Automação finalizada com sucesso")


def main() -> None:
    configurar_logging()
    executar_automacao()


if __name__ == "__main__":
    main()

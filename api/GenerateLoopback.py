import requests
import json
import random
import os
from dotenv import load_dotenv
import logging

# Configuration de la journalisation
logging.basicConfig(level=logging.INFO)

load_dotenv()
TOKEN = os.getenv("TOKEN")
NETBOX_API_URL = os.getenv("NETBOX_API_URL")
API_URL_IPAM = f"{NETBOX_API_URL}ipam/ip-addresses/"

HEADERS = {
    'Authorization': f'Token {TOKEN}',
    'Content-Type': 'application/json',
    'Accept': 'application/json',
}


def generate_ipv6_suffix(prefix, byte_count):
    """
    Génère un suffixe IPv6 aléatoire pour un préfixe donné.
    :param prefix: Préfixe IPv6.
    :param byte_count: Nombre de bytes à générer.
    :return: Adresse IPv6 complète.
    """
    suffix = "".join(random.choice("0123456789abcdef") for _ in range(byte_count))
    return f"{prefix[:-byte_count]}:{suffix}"


def make_netbox_request(url, method="get", data=None, params=None):
    """
    Effectue une requête HTTP à NetBox.
    :param url: URL de l'API NetBox.
    :param method: Méthode HTTP ('get' ou 'post').
    :param data: Données pour la requête POST.
    :param params: Paramètres pour la requête GET.
    :return: Réponse de la requête.
    """
    try:
        if method == "get":
            response = requests.get(url, headers=HEADERS, params=params)
        else:
            response = requests.post(url, headers=HEADERS, json=data)

        response.raise_for_status()
        return response.json()
    except requests.HTTPError as http_err:
        logging.error(f"HTTP error occurred: {http_err}")
    except Exception as err:
        logging.error(f"An error occurred: {err}")

    return None


def check_ip_exist_in_netbox(ip_address):
    """
    Vérifie si une adresse IP existe dans NetBox.
    :param ip_address: Adresse IPv6 à vérifier.
    :return: Booléen indiquant si l'adresse existe.
    """
    params = {"address": ip_address}
    response = make_netbox_request(API_URL_IPAM, params=params)

    if response and response["count"] > 0:
        return True
    return False


def create_ip_in_netbox(ip_address):
    """
    Crée une adresse IP dans NetBox.
    :param ip_address: Adresse IPv6 à créer.
    """
    data = {
        "address": ip_address,
        "description": "Description de l'adresse IPv6",
        "custom_fields": {"Pubkey": "Votre clé publique", "Userid": 5}
    }
    response = make_netbox_request(API_URL_IPAM, method="post", data=data)

    if response:
        logging.info(f"L'adresse IPv6 {ip_address} a été créée avec succès dans NetBox.")


def create_loopback():
    """
    Crée une adresse loopback qui n'existe pas encore dans NetBox.
    :return: Adresse loopback créée.
    """
    while True:
        temp_ip = generate_ipv6_suffix("2a13:79c0:ffff:fefe::/64", 4)
        if not check_ip_exist_in_netbox(temp_ip):
            create_ip_in_netbox(temp_ip)
            return temp_ip


loopback = create_loopback()
print(f"{loopback}/128")

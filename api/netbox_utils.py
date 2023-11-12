import requests
import json
import random

NETBOX_API_URL = 'https://netbox.360p.kube.kittenconnect.net/api/'
HEADERS = {
    'Authorization': 'Token ',
    'Content-Type': 'application/json',
    'Accept': 'application/json',
}
# Vous pouvez ajouter d'autres fonctions utiles ici (spoiler: c'est mieux)
## Generate Loopback IP address start 2a13:79c0:ffff:fefe::/64 pour faire un /128  en hexa exemple 	2a13:79c0:ffff:fefe::b47d/128
def generate_short_ipv6_address():
    prefix = "2a13:79c0:ffff:fefe::/64"

    # Générez un suffixe aléatoire de 4 caractères hexadécimaux pour /128
    suffix = "".join(random.choice("0123456789abcdef") for _ in range(4))

    # Concaténez le préfixe et le suffixe pour former l'adresse IPv6 complète
    ipv6_address = f"{prefix[:-4]}:{suffix}/128"

    return ipv6_address

def checkIpinNetbox(ip_address):
    params = {
        "address": ip_address,
    }
    apiurl=NETBOX_API_URL+"ipam/ip-addresses/"
    try:
        response = requests.get(apiurl, headers=HEADERS, params=params)

        if response.status_code == 200:
            ip_data = response.json()
            if ip_data["count"] > 0:
                return True  # L'adresse IPv6 existe déjà dans NetBox
            else:
                return False  # L'adresse IPv6 n'existe pas dans NetBox
        else:
            print("Erreur lors de la recherche de l'adresse IPv6 dans NetBox.")
            print(response.text)
            return False
    except Exception as e:
        print(f"Une erreur s'est produite : {str(e)}")
        return False

def CreateIpinNetbox(ip_address):
    apiurl=NETBOX_API_URL+"ipam/ip-addresses/"
    data = {
        "address": ip_address,
        "description": "Description de l'adresse IPv6",
        "custom_fields": {
            "Pubkey": "Votre clé publique",  
            "Userid": 5
        }  
    }

    try:
        response = requests.post(apiurl, headers=HEADERS, json=data)

        if response.status_code == 201:
            print(f"L'adresse IPv6 {ip_address} a été créée avec succès dans NetBox.")
        else:
            print("Erreur lors de la création de l'adresse IPv6 dans NetBox.")
            print(response.text)
    except Exception as e:
        print(f"Une erreur s'est produite : {str(e)}")

def generate_or_create_ipv6_address():
    while True:
        ipv6_address = generate_short_ipv6_address()
        if not checkIpinNetbox(ipv6_address):
            CreateIpinNetbox(ipv6_address)
            return ipv6_address
test =generate_or_create_ipv6_address()
print(test)


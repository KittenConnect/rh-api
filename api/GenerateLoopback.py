import requests
import json
import random
import os
from dotenv import load_dotenv
load_dotenv()
TOKEN = os.getenv("TOKEN")
NETBOX_API_URL = os.getenv("NETBOX_API_URL")
HEADERS = {
    'Authorization': f'Token {TOKEN}',
    'Content-Type': 'application/json',
    'Accept': 'application/json',
}

print(HEADERS)
def GeneratePrefixIpV6(prefix, Byte):

    suffix = "".join(random.choice("0123456789abcdef") for _ in range(4))
    ipv6_address = f"{prefix[:-Byte]}:{suffix}"
    # Concaténez le préfixe et le suffixe pour former l'adresse IPv6 complète

    return ipv6_address

def CheckIpExistInNetbox(ip_address):
    params = {
        "address": ip_address,
    }
    apiurl=NETBOX_API_URL+"ipam/ip-addresses/"
    try:
        response = requests.get(apiurl, headers=HEADERS, params=params)

        if response.status_code == 200:
            ip_data = response.json()
            if ip_data["count"] > 0:
                return True 
            else:
                return False  
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



def CreateLoopBack():
    #Le while true permet de relancer la fonction si l'adresse existe déjà dans NetBox
    while True:
        temp = GeneratePrefixIpV6("2a13:79c0:ffff:fefe::/64", 4)
        if not CheckIpExistInNetbox(temp):
            CreateIpinNetbox(temp)
            return temp

Loopback = CreateLoopBack()
print(Loopback + "/128")
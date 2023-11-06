import requests
import json
import ipaddress
NETBOX_API_URL = 'https://netbox.360p.kube.kittenconnect.net/api/'
HEADERS = {
    'Authorization': 'Token 3f530a0bad7165b7cfcf5686f2154192d5aadc3f',
    'Content-Type': 'application/json',
    'Accept': 'application/json',
}

def get_prefix(prefix_id):
    url = f"{NETBOX_API_URL}ipam/prefixes/{prefix_id}/"
    
    response = requests.get(url, headers=HEADERS)
    
    if response.status_code == 200:
        return response.json()
    else:
        return None


def allocate_subnet(prefix_id, prefix_length, description, role):
    # Obtenir les informations sur le préfixe IPv6 à partir de son ID
    prefix_info = get_prefix(prefix_id)
    
    if prefix_info:
        # Construire l'URL pour obtenir les adresses IPv6 disponibles dans le préfixe
        available_ip_url = f"{NETBOX_API_URL}/ipam/prefixes/{prefix_id}/available-ips/"
        
        # Obtenir les adresses IPv6 disponibles dans le préfixe
        response = requests.get(available_ip_url, headers=HEADERS)
        
        if response.status_code == 200:
            reponse_json = response.json()
            
            # Sélectionner la première adresse IPv6 en /128
            if reponse_json:
                ipv6_subnet_entry = reponse_json[0]
                ipv6_subnet = ipv6_subnet_entry["address"]
                
                # Obtenir le préfixe /128
                subnet_128 = ipaddress.IPv6Network(ipv6_subnet, strict=False)
                
                # Retourner l'adresse IPv6 en /128 sous forme de chaîne
                return str(subnet_128)
    
    # Si quelque chose ne se passe pas comme prévu ou si le préfixe n'existe pas
    return None
def allocate_loopback(prefix_id, description):
    return allocate_subnet(prefix_id,"128", description, "1")

# Vous pouvez ajouter d'autres fonctions utiles ici

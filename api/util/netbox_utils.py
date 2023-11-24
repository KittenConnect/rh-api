import requests
import random
import ipaddress

NETBOX_API_URL = 'https://netbox.360p.kube.kittenconnect.net/api/'
HEADERS = {
    'Authorization': 'Token DANSTAMERELECODE ',
    'Content-Type': 'application/json',
    'Accept': 'application/json',
}


def GeneratePrefixIpV6(prefix, Byte):
    # il faut le define prefix = "2a13:79c0:ffff:fefe::/64"

    # Générez un suffixe aléatoire de 4 caractères hexadécimaux pour /128
    suffix = "".join(random.choice("0123456789abcdef") for _ in range(4))

    if Byte == 4:
        ipv6_address = f"{prefix[:-Byte]}:{suffix}"
    elif Byte == 5:
        ipv6_address = f"{prefix[:-Byte]}:{suffix}:"
    else:
        raise NotImplementedError()
    # Concaténez le préfixe et le suffixe pour former l'adresse IPv6 complète

    return ipv6_address


############
#
#  Fonction pour vérifier si l'adresse IPv6 existe déjà dans NetBox
#  return True si l'adresse IPv6 existe déjà dans NetBox
#  return False si l'adresse IPv6 n'existe pas dans NetBox
############

def CheckIpExistInNetbox(ip_address):
    params = {
        "address": ip_address,
    }
    apiurl = NETBOX_API_URL + "ipam/ip-addresses/"
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


############


def generate_or_create_ipv6_address():
    while True:
        ipv6_address = generate_short_ipv6_address()
        if not checkIpinNetbox(ipv6_address):
            CreateIpinNetbox(ipv6_address)
            return ipv6_address


def get_available_address_prefix(prefix: str, length: int):
    """
    Get all the IP addresses from Prefix IPv6
    :param prefix:
    :param length:
    :return:
    """
    try:
        ipv6_network = ipaddress.IPv6Network(f"{prefix}/{length}", strict=False)
        network_address = ipv6_network.network_address
        network_address_str = str(network_address)

        return network_address_str
    except ValueError as e:
        print(f"Error : {e}")
        return None


test = GeneratePrefixIpV6("2a13:79c0:ffff:fefe::/64", 4)  # generate loopback
test2 = GeneratePrefixIpV6("2a13:79c0:ffff:feff::/64", 4) + "/127"  # generate prefix for wireguard
test3 = GeneratePrefixIpV6("2a13:79c0:ffff:feff:b00b::/80", 5)

print(test + "/128")
print(test2)  # + "/127")
print(test3 + "/96")
TAMMERE = ipaddress.IPv6Interface(test2).network

test4fin = [*ipaddress.ip_network(TAMMERE).hosts()]
print(*test4fin)


# print(TAMERELALIST)
# 2a13:79c0:ffff:feff:b00b::/80


## class NewConnection to allow Loopback IP address, and /127 subnet for wireguard and return public key
class NewConnection:
    def __init__():
        return true

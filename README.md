# roip-wanqa
# ansible-openwrt
<a id="readme-top"></a>


<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/robergeandre/roip-wanqa">
    <img src="images/RoIPLogo.png" alt="Logo" width="200" height="200">
  </a>

<h3 align="center">Cauca RoIP WanQA</h3>

  <p align="center">
    Quality Assurance pour le RoIP. 
    <br />
    <a href="https://github.com/robergeandre/roip-wanqa"><strong>Explore the docs »</strong></a>
    <br />
    &middot;
    <a href="https://github.com/robergeandre/roip-wanqa/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
    &middot;
    <a href="https://github.com/robergeandre/roip-wanqa/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
  </p>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">A propos de RoIP WanWa</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Introduction</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>

<!-- ABOUT THE PROJECT -->
## A propos du projet

[![Product Name Screen Shot][product-screenshot]](https://example.com)

Program d'analyse de la qualité du lien Wan. 
<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- GETTING STARTED -->
## Guide de démarrage

Ce projet contient la configuration du Fabric de Cauca.


### Prérequis

Voici les presrequis pour utiliser ce projet

L'installation de Ansible Galxy est requisse pour l'utilisation des collections.

1. Installation d'Ansible Galaxy

```sh 
ansible-galaxy collection install netbox.netbox
```

2. Integration de Netbox avec Ansible

Pour permettre la communication entre Ansible et NetBox, vous avez besoin de la collection Ansible officielle et d'une bibliothèque Python.

```sh 
pip install pynetbox
```
3. Creation du token NetBox 
Ansible nécessite un jeton d'API pour s'authentifier auprès de votre instance NetBox.

* Connectez-vous à l'interface web de NetBox.
* Cliquez sur votre profil dans le coin supérieur droit et sélectionnez API Tokens.
* Cliquez sur Add a Token. Vous pouvez laisser le champ « Key » vide pour laisser NetBox en générer un automatiquement.
* Sauvegardez la clé générée. Ne partagez pas ce jeton

4. Configurer les variables d'enviroennement pour NetBox 

```sh 
export NETBOX_API=https://netbox.cauca.ca
export NETBOX_TOKEN=[api_token]
```

Script pour automatiser l'installation
```sh 
echo 'export NETBOX_API="https://netbox.cauca.ca"' >> ~/.bashrc
echo 'export NETBOX_TOKEN="[api_token]"' >> ~/.bashrc
source ~/.bashrc
```

<!-- USAGE EXAMPLES -->
## Utilisation

Le code est integre dans le code du projet 'roip-wanqa'.

```sh
ansible-playbook ./playbooks/build-images.yml -K --ask-vault-pass -vvvv
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>


See the [open issues](https://github.com/github_username/ansible-fabric/issues) for a full list of proposed features (and known issues).

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Top contributors:

<a href="https://github.com/github_username/ansible-fabric/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=github_username/ansible-fabric" alt="contrib.rocks image" />
</a>



<!-- LICENSE -->
## License

Distributed under the project_license. See `LICENSE.txt` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

Your Name - [@twitter_handle](https://twitter.com/twitter_handle) - email@email_client.com

Project Link: [https://github.com/github_username/ansible-fabric](https://github.com/github_username/ansible-fabric)

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/github_username/ansible-fabric.svg?style=for-the-badge
[contributors-url]: https://github.com/github_username/ansible-fabric/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/github_username/ansible-fabric.svg?style=for-the-badge
[forks-url]: https://github.com/github_username/ansible-fabric/network/members
[stars-shield]: https://img.shields.io/github/stars/github_username/ansible-fabric.svg?style=for-the-badge
[stars-url]: https://github.com/github_username/ansible-fabric/stargazers
[issues-shield]: https://img.shields.io/github/issues/github_username/ansible-fabric.svg?style=for-the-badge
[issues-url]: https://github.com/github_username/ansible-fabric/issues
[license-shield]: https://img.shields.io/github/license/github_username/ansible-fabric.svg?style=for-the-badge
[license-url]: https://github.com/github_username/ansible-fabric/blob/master/LICENSE.txt
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[linkedin-url]: https://linkedin.com/in/linkedin_username
[product-screenshot]: images/screenshot.png
<!-- Shields.io badges. You can a comprehensive list with many more badges at: https://github.com/inttter/md-badges -->

## Application Open Source utilise dans ce projet ##
[![Markdown](https://img.shields.io/badge/Markdown-%23000000.svg?logo=markdown&logoColor=white)](#)
[![Python](https://img.shields.io/badge/Python-3776AB?logo=python&logoColor=fff)](#)
[![TOML](https://img.shields.io/badge/TOML-9C4121?logo=toml&logoColor=fff)](#)
[![YAML](https://img.shields.io/badge/YAML-CB171E?logo=yaml&logoColor=fff)](#)
[![Debian](https://img.shields.io/badge/Debian-D60000?logo=debian&logoColor=fff)](#)
[![Ansible](https://img.shields.io/badge/Ansible-FF8F00?logo=ansible&logoColor=fff)](#)
[![Git](https://img.shields.io/badge/Git-F05032?logo=git&logoColor=fff)](#)

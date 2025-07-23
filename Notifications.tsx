import React from 'react';
import { createRoot } from 'react-dom/client';
import { createPortal } from 'react-dom';
import { Zoom, toast, ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css'; // Import des styles nécessaires
import '../assets/styles/toast-custom.css'; // Import des styles personnalisés

export function ToastPortalContainer() {
	return createPortal(
		<ToastContainer
			position="bottom-right"
			autoClose={1500}
			pauseOnHover={true}
			closeOnClick={true}
			newestOnTop={true}
			closeButton={false}
			limit={3}
			transition={Zoom}
			style={{ userSelect: 'none' }}
		/>,
		document.body
	);
}

class ToastNotification {
	// Tracker pour les modals ouverts
	private static openModals: Set<() => void> = new Set();

	// Génère un ID unique basé sur le texte du message
	private static generateToastId(message: string, type: string): string {
		// Nettoie le message pour créer un ID stable
		const cleanMessage = message.replace(/[^a-zA-Z0-9]/g, '').toLowerCase();
		return `${type}-${cleanMessage.substring(0, 50)}`;
	}

	static success(message: string, options = {}) {
		const toastId = ToastNotification.generateToastId(message, 'success');

		// Vérifie si une notification avec ce contenu existe déjà
		if (toast.isActive(toastId)) {
			// Met à jour la notification existante et remet le timer à zéro
			toast.update(toastId, {
				render: ToastNotification.formatMessage(message),
				type: 'success',
				className: 'toast-simple',
				...options
			});
			return toastId;
		}

		toast.success(ToastNotification.formatMessage(message), {
			toastId,
			className: 'toast-simple',
			...options
		});
		return toastId;
	}

	static info(message: string, options = {}) {
		const toastId = ToastNotification.generateToastId(message, 'info');

		if (toast.isActive(toastId)) {
			// Met à jour la notification existante et remet le timer à zéro
			toast.update(toastId, {
				render: ToastNotification.formatMessage(message),
				type: 'info',
				className: 'toast-simple',
				...options
			});
			return toastId;
		}

		toast.info(ToastNotification.formatMessage(message), {
			toastId,
			className: 'toast-simple',
			...options
		});
		return toastId;
	}

	static warn(message: string, options = {}) {
		const toastId = ToastNotification.generateToastId(message, 'warn');

		if (toast.isActive(toastId)) {
			// Met à jour la notification existante et remet le timer à zéro
			toast.update(toastId, {
				render: ToastNotification.formatMessage(message),
				type: 'warning',
				className: 'toast-simple',
				...options
			});
			return toastId;
		}

		toast.warn(ToastNotification.formatMessage(message), {
			toastId,
			className: 'toast-simple',
			...options
		});
		return toastId;
	}

	static error(message: string, options = {}) {
		const toastId = ToastNotification.generateToastId(message, 'error');

		if (toast.isActive(toastId)) {
			// Met à jour la notification existante et remet le timer à zéro
			toast.update(toastId, {
				render: ToastNotification.formatMessage(message),
				type: 'error',
				className: 'toast-simple',
				...options
			});
			return toastId;
		}

		toast.error(ToastNotification.formatMessage(message), {
			toastId,
			className: 'toast-simple',
			...options
		});
		return toastId;
	}

	static promise(promise: Promise<unknown> | (() => Promise<unknown>), successMessage: string, errorMessage: string, options = {}) {
		toast.promise(promise, {
			pending: "En cours...",
			success: successMessage,
			error: errorMessage,
		}, { ...options });
	}

	// Méthode pour fermer une notification spécifique
	static dismiss(toastId: string) {
		toast.dismiss(toastId);
	}

	// Utiliser le type React.ReactElement pour le typage
	private static formatMessage(message: string): React.ReactElement {
		const lines = message.split('\n');
		return (
			<div style={{ display: 'flex', flexDirection: 'column' }}>
				{lines.map((line, idx) => (
					<div key={idx}>{line}</div>
				))}
			</div>
		);
	}

	static confirm(
		message: string,
		_options = {}
	): Promise<boolean> {
		return new Promise((resolve) => {
			const handleAccept = () => {
				closeModal();
				resolve(true);
			};

			const handleRefuse = () => {
				closeModal();
				resolve(false);
			};

			// Créer le contenu du modal
			const content = (
				<div style={{
					backgroundColor: 'white',
					padding: '40px',
					borderRadius: '12px',
					boxShadow: '0 20px 60px rgba(0, 0, 0, 0.3)',
					maxWidth: '500px',
					width: '90%',
					margin: '20px',
					textAlign: 'center'
				}}>
					<div style={{
						fontSize: '18px',
						marginBottom: '30px',
						color: '#333',
						lineHeight: '1.5'
					}}>
						{ToastNotification.formatMessage(message)}
					</div>
					<div style={{
						display: 'flex',
						gap: '15px',
						justifyContent: 'center',
						flexWrap: 'wrap'
					}}>
						<button
							onClick={handleAccept}
							style={{
								padding: '12px 30px',
								backgroundColor: '#4CAF50',
								color: 'white',
								border: 'none',
								borderRadius: '8px',
								cursor: 'pointer',
								fontSize: '16px',
								fontWeight: '600',
								minWidth: '120px',
								transition: 'all 0.2s ease',
								boxShadow: '0 2px 8px rgba(76, 175, 80, 0.3)'
							}}
							onMouseEnter={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#45a049';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(-1px)';
							}}
							onMouseLeave={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#4CAF50';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(0)';
							}}
						>
							Accepter
						</button>
						<button
							onClick={handleRefuse}
							style={{
								padding: '12px 30px',
								backgroundColor: '#f44336',
								color: 'white',
								border: 'none',
								borderRadius: '8px',
								cursor: 'pointer',
								fontSize: '16px',
								fontWeight: '600',
								minWidth: '120px',
								transition: 'all 0.2s ease',
								boxShadow: '0 2px 8px rgba(244, 67, 54, 0.3)'
							}}
							onMouseEnter={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#da190b';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(-1px)';
							}}
							onMouseLeave={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#f44336';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(0)';
							}}
						>
							Refuser
						</button>
					</div>
				</div>
			);

			// Utiliser la fonction générale pour créer le modal
			const { closeModal } = ToastNotification.createFullScreenModal(content);
		});
	}

	static dismissAll() {
		// Fermer tous les toasts
		toast.dismiss();
		
		// Fermer tous les modals ouverts
		ToastNotification.openModals.forEach(closeModalFn => {
			try {
				closeModalFn();
			} catch (error) {
				console.error('Error closing modal:', error);
			}
		});
		
		// Vider la liste des modals
		ToastNotification.openModals.clear();
	}

	static cancel(
		message: string,
		_options = {}
	): Promise<boolean> {
		return new Promise((resolve) => {
			const handleAccept = () => {
				closeModal();
				resolve(true);
			};

			// Créer le contenu du modal
			const content = (
				<div style={{
					backgroundColor: 'white',
					padding: '40px',
					borderRadius: '12px',
					boxShadow: '0 20px 60px rgba(0, 0, 0, 0.3)',
					maxWidth: '500px',
					width: '90%',
					margin: '20px',
					textAlign: 'center'
				}}>
					<div style={{
						fontSize: '18px',
						marginBottom: '30px',
						color: '#333',
						lineHeight: '1.5'
					}}>
						{ToastNotification.formatMessage(message)}
					</div>
					<div style={{
						display: 'flex',
						gap: '15px',
						justifyContent: 'center',
						flexWrap: 'wrap'
					}}>
						<button
							onClick={handleAccept}
							style={{
								padding: '12px 30px',
								backgroundColor: '#4CAF50',
								color: 'white',
								border: 'none',
								borderRadius: '8px',
								cursor: 'pointer',
								fontSize: '16px',
								fontWeight: '600',
								minWidth: '120px',
								transition: 'all 0.2s ease',
								boxShadow: '0 2px 8px rgba(76, 175, 80, 0.3)'
							}}
							onMouseEnter={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#45a049';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(-1px)';
							}}
							onMouseLeave={(e) => {
								(e.currentTarget as HTMLButtonElement).style.backgroundColor = '#4CAF50';
								(e.currentTarget as HTMLButtonElement).style.transform = 'translateY(0)';
							}}
						>
							Annuler
						</button>
					</div>
				</div>
			);

			// Utiliser la fonction générale pour créer le modal
			const { closeModal } = ToastNotification.createFullScreenModal(content);
		});
	}

	// Fonction générale pour créer un modal plein écran
	private static createFullScreenModal(content: React.ReactElement): {
		modalDiv: HTMLDivElement;
		root: ReturnType<typeof createRoot>;
		closeModal: () => void;
	} {
		// Créer le modal en plein écran
		const modalDiv = document.createElement('div');
		modalDiv.style.cssText = `
			position: fixed;
			top: 0;
			left: 0;
			width: 100vw;
			height: 100vh;
			background-color: rgba(0, 0, 0, 0.7);
			display: flex;
			justify-content: center;
			align-items: center;
			z-index: 9999;
			backdrop-filter: blur(5px);
		`;

		// Créer une root React pour le modal
		const root = createRoot(modalDiv);

		const closeModal = () => {
			root.unmount();
			document.body.removeChild(modalDiv);
			document.body.style.overflow = ''; // Restaurer le scroll
		};

		// Bloquer le scroll de la page
		document.body.style.overflow = 'hidden';

		// Rendu du contenu dans le modal
		root.render(content);

		// Ajouter le modal au body
		document.body.appendChild(modalDiv);

		// Empêcher la fermeture en cliquant en dehors du modal
		modalDiv.addEventListener('click', (e) => {
			if (e.target === modalDiv) {
				// Ne rien faire - l'utilisateur doit obligatoirement choisir
				e.preventDefault();
				e.stopPropagation();
			}
		});

		// Empêcher la fermeture avec Echap
		const handleKeyDown = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				e.preventDefault();
				e.stopPropagation();
			}
		};
		document.addEventListener('keydown', handleKeyDown);

		// Créer une version du closeModal qui nettoie les event listeners
		const originalCloseModal = closeModal;
		const enhancedCloseModal = () => {
			document.removeEventListener('keydown', handleKeyDown);
			ToastNotification.openModals.delete(enhancedCloseModal); // Retirer de la liste
			originalCloseModal();
		};

		// Ajouter le modal à la liste des modals ouverts
		ToastNotification.openModals.add(enhancedCloseModal);

		return {
			modalDiv,
			root,
			closeModal: enhancedCloseModal
		};
	}

	static alert(
		message: string,
		toastId: string,
		options = {}
	): Promise<void> {
		return new Promise((resolve) => {
			const content = (

				<div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
					{ToastNotification.formatMessage(message)}
					<div style={{ display: 'flex', justifyContent: 'center' }}>
						<button
							onClick={() => {
								resolve();
								toast.dismiss(toastId);
							}}
							style={{
								padding: '8px 20px',
								backgroundColor: '#2196F3',
								color: 'white',
								border: 'none',
								borderRadius: '4px',
								cursor: 'pointer',
								fontSize: '14px',
								fontWeight: '500',
								display: 'flex',
								justifyContent: 'center',
							}}
						>
							OK
						</button>
					</div>
				</div>
			);

			// Vérifier si une notification avec cet ID existe déjà
			if (toast.isActive(toastId)) {
				// Mettre à jour le contenu de la notification existante
				toast.update(toastId, {
					render: content,
					autoClose: false,
					closeOnClick: false,
					className: 'toast-with-buttons',
					...options
				});
			} else {
				// Créer une nouvelle notification avec l'ID spécifié
				toast(content, {
					toastId: toastId,
					autoClose: false,
					closeOnClick: false,
					className: 'toast-with-buttons',
					...options
				});
			}
		});
	}


}

// Exporter à la fois la classe et le composant ToastContainer
export { ToastContainer };
export default ToastNotification;

// Examples of usage

// Basic notifications (retournent maintenant un ID)
// const successId = ToastNotification.success("Operation successful!");
// const infoId = ToastNotification.info("Here's some information for you.");
// const warnId = ToastNotification.warn("Warning: This action may have consequences.");
// const errorId = ToastNotification.error("An error occurred!");

// Notifications avec ID automatique - mise à jour intelligente
// ToastNotification.success("Same message"); // Première notification
// ToastNotification.success("Same message"); // Met à jour l'existante et remet le timer à zéro
// ToastNotification.success("Different message"); // Nouvelle notification

// With custom options
// const customId = ToastNotification.success("Custom autoclose!", { autoClose: 5000 });
// ToastNotification.info("Custom position!", { position: 'top-center' });

// Fermer une notification spécifique avec son ID
// const notifId = ToastNotification.error("Error message");
// setTimeout(() => ToastNotification.dismiss(notifId), 3000);

// Promise example
// const asyncOperation = async () => {
//   return new Promise((resolve, reject) => {
//     // Simulate API call
//     setTimeout(() => {
//       const success = Math.random() > 0.5;
//       if (success) {
//         resolve("Data received");
//       } else {
//         reject("Network error");
//       }
//     }, 2000);
//   });
// };
// 
// ToastNotification.promise(
//   asyncOperation,
//   "Data successfully loaded!",
//   "Failed to load data"
// );

// Confirm dialog example
// ToastNotification.confirm("Are you sure you want to proceed?")
//   .then((result) => {
//     if (result) {

// Alert dialog example (just OK button)
// ToastNotification.alert("This is an important message!")
//   .then(() => {
//     console.log("User clicked OK");

// Alert dialog with custom ID (updates existing notification with same ID)
// ToastNotification.alertWithId("Updated message!", "my-custom-id")
//   .then(() => {
//     console.log("User clicked OK on updated notification");